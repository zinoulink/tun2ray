# Usage
## Windows
netstat -nr
netsh interface ip add route 0.0.0.0/0 mellow-tap0 10.0.85.1 metric=0 store=active
route add 0.0.0.0 mask 0.0.0.0 10.0.85.1 metric 1
.\tun2ray.exe -tunName mellow-tap0 -tunAddr 10.0.85.2 -tunGw 10.0.85.1 -tunDns 114.114.114.114,8.8.8.8 -sendThrough 192.168.0.101:0 -config config.json
route delete 0.0.0.0 mask 0.0.0.0 10.0.85.1 

# Build
go get -d ./...
## Android
gomobile bind -target=android -o build/tun2ray.aar fipn.xyz/tun2ray/android
gomobile bind -a -ldflags '-s -w' -target=android -o build/tun2ray.aar fipn.xyz/tun2ray/android
## Windows
env GOOS=windows GOARCH=386 CGO_ENABLED=1 go build -o build/tun2ray.exe
tun2ray.dll should be compiled in 32 bit because GTK# is available  only in 32 bit
pacman -S mingw-w64-i686-gcc
change path to mingw32
env GOOS=windows GOARCH=386 CGO_ENABLED=1 go build -ldflags="-H windowsgui" -o build/tun2ray.dll -buildmode=c-shared fipn.xyz/tun2ray/windows

# Update modules
go get -u