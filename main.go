package main

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"fipn.xyz/tun2ray/d"
	"fipn.xyz/tun2ray/v2ray"
	vcore "v2ray.com/core"
	vproxyman "v2ray.com/core/app/proxyman"
	vbytespool "v2ray.com/core/common/bytespool"

	"github.com/eycorsican/go-tun2socks/core"
	"github.com/eycorsican/go-tun2socks/tun"
)

type cmdArgs struct {
	TunName              *string
	TunAddr              *string
	TunGw                *string
	TunMask              *string
	TunDNS               *string
	Config               *string
	SniffingType         *string
	UDPTimeout           *time.Duration
	DNSFallback          *bool
	ExceptionApps        *string
	ExceptionSendThrough *string
}

const (
	MTU = 1500
)

var args = new(cmdArgs)
var lwipWriter io.Writer

func main() {
	args.TunName = flag.String("tunName", "mellow-tap0", "TUN interface name")
	args.TunAddr = flag.String("tunAddr", "10.0.85.2", "TUN interface address")
	args.TunGw = flag.String("tunGw", "10.0.85.1", "TUN interface gateway")
	args.TunMask = flag.String("tunMask", "255.255.255.0", "TUN interface netmask, it should be a prefixlen (a number) for IPv6 address")
	args.TunDNS = flag.String("tunDns", "114.114.114.114,8.8.8.8", "DNS resolvers for TUN interface (only need on Windows)")
	args.Config = flag.String("config", "config.json", "Config file for v2ray, in JSON format, and note that routing in v2ray could not violate routes in the routing table")
	args.SniffingType = flag.String("sniffingType", "http,tls", "Enable domain sniffing for specific kind of traffic in v2ray")
	args.ExceptionApps = flag.String("exceptionApps", "tun2ray.exe", "Exception app list separated by commas")
	args.ExceptionSendThrough = flag.String("sendThrough", "192.168.0.101:0", "Exception send through address")

	flag.Parse()

	if args.UDPTimeout == nil {
		args.UDPTimeout = flag.Duration("udpTimeout", 1*time.Minute, "UDP session timeout")
	}

	// Open the tun device.
	dnsServers := strings.Split(*args.TunDNS, ",")
	tunDev, err := tun.OpenTunDevice(*args.TunName, *args.TunAddr, *args.TunGw, *args.TunMask, dnsServers, false)
	if err != nil {
		log.Fatalf("failed to open tun device: %v", err)
	}

	// Setup TCP/IP stack.
	lwipWriter := core.NewLWIPStack().(io.Writer)

	startV2Ray(*args.Config, *args.SniffingType, *args.ExceptionApps, *args.ExceptionSendThrough)

	// Register an output callback to write packets output from lwip stack to tun
	// device, output function should be set before input any packets.
	core.RegisterOutputFn(func(data []byte) (int, error) {
		return tunDev.Write(data)
	})

	// Copy packets from tun device to lwip stack, it's the main loop.
	go func() {
		_, err := io.CopyBuffer(lwipWriter, tunDev, make([]byte, MTU))
		if err != nil {
			log.Fatalf("copying data failed: %v", err)
		}
	}()

	log.Println("Running tun2ray")
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGHUP)
	<-osSignals
}

func startV2Ray(configFile string, sniffingType string, exceptionApps string, exceptionSendThrough string) {
	core.SetBufferPool(vbytespool.GetPool(core.BufSize))

	configBytes, err := ioutil.ReadFile(*args.Config)
	if err != nil {
		log.Fatalf("invalid vconfig file")
	}

	var validSniffings []string
	sniffings := strings.Split(sniffingType, ",")
	for _, s := range sniffings {
		if s == "http" || s == "tls" {
			validSniffings = append(validSniffings, s)
		}
	}

	v, err := vcore.StartInstance("json", configBytes)
	if err != nil {
		log.Fatalf("start V instance failed: %v", err)
	}

	sniffingConfig := &vproxyman.SniffingConfig{
		Enabled:             true,
		DestinationOverride: validSniffings,
	}
	if len(validSniffings) == 0 {
		sniffingConfig.Enabled = false
	}

	ctx := vproxyman.ContextWithSniffingConfig(context.Background(), sniffingConfig)

	v2rayTCPConnHandler := v2ray.NewTCPHandler(ctx, v)
	v2rayUDPConnHandler := v2ray.NewUDPHandler(ctx, v, *args.UDPTimeout)

	sendThrough, err := net.ResolveTCPAddr("tcp", exceptionSendThrough)
	if err != nil {
		log.Fatalf("invalid exception send through address: %v", err)
	}
	apps := strings.Split(exceptionApps, ",")

	tcpHandler := d.NewTCPHandler(v2rayTCPConnHandler, apps, sendThrough)
	udpHandler := d.NewUDPHandler(v2rayUDPConnHandler, apps, sendThrough, *args.UDPTimeout)

	core.RegisterTCPConnHandler(tcpHandler)
	core.RegisterUDPConnHandler(udpHandler)
}
