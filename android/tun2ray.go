package tun2ray

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"fipn.xyz/tun2ray/dnsfallback"
	"fipn.xyz/tun2ray/v2ray"
	vcore "v2ray.com/core"
	vproxyman "v2ray.com/core/app/proxyman"
	vbytespool "v2ray.com/core/common/bytespool"

	"github.com/eycorsican/go-tun2socks/core"
)

var lwipStack core.LWIPStack
var v *vcore.Instance
var isStopped = false

// Start sets up lwIP stack, starts a V2Ray instance and registers the instance as the
// connection handler for tun2socks.
func Start(fd int, Config string, IsUDPEnabled bool, MTU int) {
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
	configBytes := []byte(Config)

	// Start the V2Ray instance.
	v, err = vcore.StartInstance("json", configBytes)
	if err != nil {
		log.Fatalf("start V instance failed: %v", err)
	}

	// Configure sniffing settings for traffic coming from tun2socks.
	var validSniffings []string
	sniffings := strings.Split("http,tls", ",")
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

	// MakeTunFile returns an os.File object from a TUN file descriptor `fd`.
	tun := os.NewFile(uintptr(fd), "")
	// Write IP packets back to TUN.
	core.RegisterOutputFn(func(data []byte) (int, error) {
		return tun.Write(data)
	})

	// Register tun2socks connection handlers.
	TCPConnHandler := v2ray.NewTCPHandler(ctx, v)
	var UDPConnHandler core.UDPConnHandler
	if IsUDPEnabled {
		UDPConnHandler = v2ray.NewUDPHandler(ctx, v, 1*time.Minute)
	} else {
		UDPConnHandler = dnsfallback.NewUDPHandler()
	}

	core.RegisterTCPConnHandler(TCPConnHandler)
	core.RegisterUDPConnHandler(UDPConnHandler)

	isStopped = false

	if lwipStack == nil {
		// Setup the lwIP stack.
		lwipStack = core.NewLWIPStack()
	}

	// ProcessInputPackets reads packets from a TUN device `tun` and writes them to `lwipStack`
	// It's the main loop
	buffer := make([]byte, MTU)
	for !isStopped {
		len, err := tun.Read(buffer)
		if err != nil {
			log.Println("Failed to read packet from TUN: %v", err)
			continue
		}
		if len == 0 {
			log.Println("Read EOF from TUN")
			continue
		}
		if lwipStack != nil {
			lwipStack.Write(buffer)
		}
	}
}

// Stop V2Ray, close lwipStack
func Stop() {
	isStopped = true
	if lwipStack != nil {
		err := lwipStack.Close()
		if err != nil {
			fmt.Println(err)
		}
		lwipStack = nil
	}
	if v != nil {
		err := v.Close()
		if err != nil {
			fmt.Println(err)
		}
		v = nil
	}
}
