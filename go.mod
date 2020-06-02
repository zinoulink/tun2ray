module fipn.xyz/tun2ray

require (
	github.com/eycorsican/go-tun2socks v1.16.9
	github.com/golang/mock v1.3.1 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/miekg/dns v1.1.14 // indirect
	github.com/refraction-networking/utls v0.0.0-20200601200209-ada0bb9b38a0 // indirect
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8 // indirect
	go.starlark.net v0.0.0-20200519165436-0aa95694c768 // indirect
	golang.org/x/crypto v0.0.0-20200510223506-06a226fb4e37 // indirect
	golang.org/x/mobile v0.0.0-20200329125638-4c31acba0007 // indirect
	golang.org/x/net v0.0.0-20200528225125-3c3fba18258b // indirect
	golang.org/x/sys v0.0.0-20200523222454-059865788121
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20200601130524-0f60399e6634 // indirect
	google.golang.org/grpc v1.29.1 // indirect
	v2ray.com/core v4.19.1+incompatible
)

replace v2ray.com/core => github.com/v2ray/v2ray-core v4.23.2+incompatible

go 1.14
