package main

// #cgo CFLAGS: -g -Wall
// #include <stdlib.h>

import "C"
import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"fipn.xyz/tun2ray/d"
	"fipn.xyz/tun2ray/tun"
	"fipn.xyz/tun2ray/v2ray"

	vcore "v2ray.com/core"
	vproxyman "v2ray.com/core/app/proxyman"
	vbytespool "v2ray.com/core/common/bytespool"

	"github.com/eycorsican/go-tun2socks/core"
)

func main() {}

var lwipStack core.LWIPStack
var v *vcore.Instance
var isStopped = false
var tunDev io.ReadWriteCloser
var err error

//export StartTun2Ray
func StartTun2Ray(tunName *C.char, tunAddr *C.char, tunGw *C.char, tunMask *C.char,
	tunDNS *C.char, config *C.char, exceptionApps *C.char, sendThrough *C.char, MTU int) {

	// Coverte parameters to Go string
	TunName := C.GoString(tunName)
	TunAddr := C.GoString(tunAddr)
	TunGw := C.GoString(tunGw)
	TunMask := C.GoString(tunMask)
	TunDNS := C.GoString(tunDNS)
	Config := C.GoString(config)
	ExceptionApps := C.GoString(exceptionApps)
	SendThrough := C.GoString(sendThrough)
	SniffingType := "http,tls"
	UDPTimeout := 1 * time.Minute

	// Open the tun device.
	dnsServers := strings.Split(TunDNS, ",")
	tunDev, err = tun.OpenTunDevice(TunName, TunAddr, TunGw, TunMask, dnsServers, false)
	if err != nil {
		log.Fatalf("failed to open tun device: %v", err)
	}

	// Setup TCP/IP stack.
	lwipWriter := core.NewLWIPStack().(io.Writer)

	startV2Ray(Config, SniffingType, ExceptionApps, SendThrough, UDPTimeout)

	isStopped = false

	// Register an output callback to write packets output from lwip stack to tun
	// device, output function should be set before input any packets.
	core.RegisterOutputFn(func(data []byte) (int, error) {
		if isStopped {
			fmt.Println("tunDev is Closed")
			return 0, nil
		}
		return tunDev.Write(data)
	})

	// Copy packets from tun device to lwip stack, it's the main loop.
	go func() {
		_, err := io.CopyBuffer(lwipWriter, tunDev, make([]byte, MTU))
		if err != nil {
			log.Println("copying data failed: %v", err)
			return
		}
	}()

	log.Println("Running tun2ray")
}

//export StopTun2Ray
func StopTun2Ray() {
	isStopped = true
	// Close tun Device
	if tunDev != nil {
		tunDev.Close()
		if err != nil {
			fmt.Println(err)
		}
		tunDev = nil
	}
	// Close lwipStack
	if lwipStack != nil {
		err := lwipStack.Close()
		if err != nil {
			fmt.Println(err)
		}
		lwipStack = nil
	}
	// Close v2ray instance
	if v != nil {
		err := v.Close()
		if err != nil {
			fmt.Println(err)
		}
		v = nil
	}
	fmt.Println("Stoped")
}
func startV2Ray(config string, sniffingType string, exceptionApps string,
	exceptionSendThrough string, UDPTimeout time.Duration) {

	// Change V2ray asset path to the current path
	// to access geosite.dat & geoipdat
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	os.Setenv("v2ray.location.asset", path)

	// Share the buffer pool.
	core.SetBufferPool(vbytespool.GetPool(core.BufSize))

	// Converte config to bytes.
	configBytes := []byte(config)

	// Start the V2Ray instance.
	v, err = vcore.StartInstance("json", configBytes)
	if err != nil {
		log.Fatalf("start V instance failed: %v", err)
	}

	// Configure sniffing settings for traffic coming from tun2socks.
	var validSniffings []string
	sniffings := strings.Split(sniffingType, ",")
	for _, s := range sniffings {
		if s == "http" || s == "tls" {
			validSniffings = append(validSniffings, s)
		}
	}
	sniffingConfig := &vproxyman.SniffingConfig{
		Enabled:             true,
		DestinationOverride: validSniffings,
	}
	if len(validSniffings) == 0 {
		sniffingConfig.Enabled = false
	}
	ctx := vproxyman.ContextWithSniffingConfig(context.Background(), sniffingConfig)

	// Create v2ray handlers.
	v2rayTCPConnHandler := v2ray.NewTCPHandler(ctx, v)
	v2rayUDPConnHandler := v2ray.NewUDPHandler(ctx, v, UDPTimeout)

	// Resolve the gateway address.
	sendThrough, err := net.ResolveTCPAddr("tcp", exceptionSendThrough)
	if err != nil {
		log.Fatalf("invalid exception send through address: %v", err)
	}
	// Prepare the apps list.
	apps := strings.Split(exceptionApps, ",")

	// Create d handlers
	tcpHandler := d.NewTCPHandler(v2rayTCPConnHandler, apps, sendThrough)
	udpHandler := d.NewUDPHandler(v2rayUDPConnHandler, apps, sendThrough, UDPTimeout)

	// Register tun2socks connection handlers.
	core.RegisterTCPConnHandler(tcpHandler)
	core.RegisterUDPConnHandler(udpHandler)
}