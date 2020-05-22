package tun2ray

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"fipn.xyz/tun2ray/dnsfallback"
	"fipn.xyz/tun2ray/v2ray"
	vcore "v2ray.com/core"
	vproxyman "v2ray.com/core/app/proxyman"
	vbytespool "v2ray.com/core/common/bytespool"
	vinternet "v2ray.com/core/transport/internet"

	"github.com/eycorsican/go-tun2socks/core"
)

// VpnService should be implemented in Java/Kotlin.
type VpnService interface {
	// Protect is just a proxy to the VpnService.protect() method.
	// See also: https://developer.android.com/reference/android/net/VpnService.html#protect(int)
	Protect(fd int) bool
}

var lwipStack core.LWIPStack
var v *vcore.Instance
var isStopped = false

// Start sets up lwIP stack, starts a V2Ray instance and registers the instance as the
// connection handler for tun2socks.
func Start(fd int, vpnService VpnService, ConfigFile string, IsUDPEnabled bool, MTU int) {
	// Assets
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	os.Setenv("v2ray.location.asset", path)

	// SetNonblock puts the fd in blocking or non-blocking mode.
	/*err = syscall.SetNonblock(fd, false)
	if err != nil {
		return
	}*/

	// Protect file descriptors of net connections in the VPN process to prevent infinite loop.
	// It works only with http, doesn't work with tls
	protectFd := func(s VpnService, fd int) error {
		if s.Protect(fd) {
			return nil
		} else {
			return fmt.Errorf(fmt.Sprintf("failed to protect fd %v", fd))
		}
	}
	netCtlr := func(network, address string, fd uintptr) error {
		return protectFd(vpnService, int(fd))
	}
	vinternet.RegisterDialerController(netCtlr)
	vinternet.RegisterListenerController(netCtlr)

	// Share the buffer pool.
	core.SetBufferPool(vbytespool.GetPool(core.BufSize))

	configBytes, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		//log.Fatalf("invalid vconfig file")
		fmt.Println(err)
	}

	// Start the V2Ray instance.
	v, err = vcore.StartInstance("json", configBytes)
	if err != nil {
		log.Fatal("start V instance failed: %v", err)
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

// StopV2Ray ...
func StopV2Ray() {
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

func init() {
	/*net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d, _ := vnet.ParseDestination(fmt.Sprintf("%v:%v", network, localDNS))
			return vinternet.DialSystem(ctx, d, nil)
		},
	}
	d := &net.Dialer{}
	d.Control = func(network, address string, c syscall.RawConn) error {
		return c.Control(func(fd uintptr) {
			// Access socket fd
		})
	}*/
}
