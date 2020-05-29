GOMOBILE=gomobile
GOBIND=$(GOMOBILE) bind
BUILDDIR=$(shell pwd)/build
IOS_ARTIFACT=$(BUILDDIR)/Tun2ray.framework
ANDROID_ARTIFACT=$(BUILDDIR)/tun2ray.aar
IOS_TARGET=ios
ANDROID_TARGET=android
LDFLAGS='-s -w'
IMPORT_PATH_IOS=fipn.xyz/tun2ray/ios
IMPORT_PATH_ANDROID=fipn.xyz/tun2ray/android

BUILD_IOS="cd $(BUILDDIR) && $(GOBIND) -a -ldflags $(LDFLAGS) -target=$(IOS_TARGET) -o $(IOS_ARTIFACT) $(IMPORT_PATH_IOS)"
BUILD_ANDROID="cd $(BUILDDIR) && $(GOBIND) -a -ldflags $(LDFLAGS) -target=$(ANDROID_TARGET) -o $(ANDROID_ARTIFACT) $(IMPORT_PATH_ANDROID)"

all: ios android

ios:
	mkdir -p $(BUILDDIR)
	eval $(BUILD_IOS)

android:
	mkdir -p $(BUILDDIR)
	eval $(BUILD_ANDROID)

clean:
	rm -rf $(BUILDDIR)

# HelloWorld
windows: 
	env GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -i -v -buildmode=c-shared -o build/tun2ray.dll fipn.xyz/tun2ray/windows
