package tun2ray

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/zinoulink/tun2ray/dnsfallback"
	"github.com/zinoulink/tun2ray/v2ray"

	vcore "v2ray.com/core"
	vproxyman "v2ray.com/core/app/proxyman"
	vbytespool "v2ray.com/core/common/bytespool"
	"v2ray.com/core/common/session"

	"github.com/eycorsican/go-tun2socks/core"
)

var lwipStack core.LWIPStack
var v *vcore.Instance
var isStopped = false

// Start sets up lwIP stack, starts a V2Ray instance and registers the instance as the
// connection handler for tun2socks.
func Start(fd int, Config string, IsUDPEnabled bool, MTU int) string {

	// Change V2ray asset path to the current path
	// to access geosite.dat & geoipdat
	path, err := os.Getwd()
	if err != nil {
		return fmt.Sprintln(err.Error())
	}
	os.Setenv("v2ray.location.asset", path)

	// Share the buffer pool.
	core.SetBufferPool(vbytespool.GetPool(core.BufSize))

	// Converte config to bytes.
	configBytes := []byte(Config)

	// Start the V2Ray instance.
	v, err = vcore.StartInstance("json", configBytes)
	if err != nil {
		return fmt.Sprintln("start V instance failed: ", err.Error())
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
	ctx := ContextWithSniffingConfig(context.Background(), sniffingConfig)

	// MakeTunFile returns an os.File object from a TUN file descriptor `fd`.
	tun := os.NewFile(uintptr(fd), "")
	// Write IP packets back to TUN.
	core.RegisterOutputFn(func(data []byte) (int, error) {
		if isStopped {
			fmt.Println("tunDev is Closed")
			return 0, nil
		}
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
	buf := make([]byte, MTU)
	go func() {
		_, err := io.CopyBuffer(lwipStack, tun, buf)
		if err != nil {
			fmt.Println("copying data failed: %v", err)
			return
		}
	}()

	fmt.Println("Running tun2ray")
	return ""
}

// Stop V2Ray, close lwipStack
func Stop() string {
	isStopped = true
	if lwipStack != nil {
		err := lwipStack.Close()
		if err != nil {
			return fmt.Sprintln(err.Error())
		}
		lwipStack = nil
	}
	if v != nil {
		err := v.Close()
		if err != nil {
			return fmt.Sprintln(err.Error())
		}
		v = nil
	}
	return ""
}

// ContextWithSniffingConfig is a wrapper of session.ContextWithContent.
// Deprecated. Use session.ContextWithContent directly.
func ContextWithSniffingConfig(ctx context.Context, c *vproxyman.SniffingConfig) context.Context {
	content := session.ContentFromContext(ctx)
	if content == nil {
		content = new(session.Content)
		ctx = session.ContextWithContent(ctx, content)
	}
	content.SniffingRequest.Enabled = c.Enabled
	content.SniffingRequest.OverrideDestinationForProtocol = c.DestinationOverride
	return ctx
}
