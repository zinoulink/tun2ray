module github.com/zinoulink/tun2ray

require (
	github.com/eycorsican/go-tun2socks v1.16.9
	github.com/golang/protobuf v1.4.3
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/refraction-networking/utls v0.0.0-20200601200209-ada0bb9b38a0 // indirect
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8 // indirect
	golang.org/x/mobile v0.0.0-20200329125638-4c31acba0007 // indirect
	golang.org/x/sys v0.0.0-20201231184435-2d18734c6014
	v2ray.com/core v4.19.1+incompatible
)

replace v2ray.com/core => github.com/v2fly/v2ray-core v4.34.0+incompatible

go 1.15
