# Usage
## Windows
netstat -nr
route add 0.0.0.0 mask 0.0.0.0 10.0.85.1 metric 0
.\tun2ray.exe -tunName mellow-tap0 -tunAddr 10.0.85.2 -tunGw 10.0.85.1 -tunDns 114.114.114.114,8.8.8.8 -sendThrough 192.168.0.101:0 -config config.json
route delete 0.0.0.0 mask 0.0.0.0 10.0.85.1 

# Build
## Android
go get -d ./...
gomobile bind -target=android -o build/tun2ray.aar fipn.xyz/tun2ray/android
gomobile bind -a -ldflags '-s -w' -target=android -o build/tun2ray.aar fipn.xyz/tun2ray/android
## Windows
go build -o build/tun2ray.exe