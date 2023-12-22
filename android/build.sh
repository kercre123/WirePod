#!/bin/bash

set -e

export ANDROID_HOME=$HOME/Android/Sdk
export CC=${ANDROID_HOME}/ndk-bundle/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android23-clang

if [[ ! $1 ]]; then
	echo "You must provide a verison (./build.sh 1.0.0)"
	exit 1
fi
if [[ ! -f $CC ]]; then
    echo "Couldn't find $CC"
    echo "You must install the Android SDK and an ndk-bundle."
    exit 1
fi
if [[ ! -f $HOME/go/bin/fyne ]]; then
    echo "Couldn't find fyne"
    echo 'This can be instaled with "go install fyne.io/fyne/v2/cmd/fyne@latest"'
    exit 1
fi
if [[ ! -f key/ks.jks ]]; then
    echo "Signing keystore not found"
    echo "Generate a key with keytool. There must be a keystore at ./key/ks.jks and a password for that at ./key/passwd"
    echo 'Ex: "keytool -genkey -v -keystore your-keystore.jks -keyalg RSA -keysize 2048 -validity 10000 -alias your-alias"'
    exit 1
fi
if [[ ! -f key/passwd ]]; then
    echo "Signing keystore not found"
    echo "Generate a key with keytool. There must be a keystore at ./key/ks.jks and a password for that at ./key/passwd"
    echo 'Ex: "keytool -genkey -v -keystore your-keystore.jks -keyalg RSA -keysize 2048 -validity 10000 -alias your-alias"'
    exit 1
fi
echo "Zipping static files and bundling..."
cd resources
echo $1 > version
zip -r static.zip .
cd ..
rm -f static.go
$HOME/go/bin/fyne bundle -o static.go resources/static.zip
export CXX=${ANDROID_HOME}/ndk-bundle/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android23-clang++
export CGO_ENABLED=1
export CGO_LDFLAGS="-L$(pwd)/built-libs/arm64/lib"
export CGO_CFLAGS="-I$(pwd)/built-libs/arm64/include"
echo "Building vessel APK..."
cd vessel
GOOS=android GOARCH=arm64 $HOME/go/bin/fyne package -os android/arm64 -appID com.kercre123.wirepod -icon ../icons/png/podfull.png --name WirePod --tags nolibopusfile --appVersion $1
cp WirePod.apk ../
cd ..
echo "Building WirePod for android/arm64..."
GOOS=android GOARCH=arm64 go build -buildmode=c-shared -o libWirePod-arm64.so -tags nolibopusfile
export CC=${ANDROID_HOME}/ndk-bundle/toolchains/llvm/prebuilt/linux-x86_64/bin/armv7a-linux-androideabi23-clang
export CXX=${ANDROID_HOME}/ndk-bundle/toolchains/llvm/prebuilt/linux-x86_64/bin/armv7a-linux-androideabi23-clang++
export CGO_ENABLED=1
export CGO_LDFLAGS="-L$(pwd)/built-libs/armv7/lib"
export CGO_CFLAGS="-I$(pwd)/built-libs/armv7/include"
echo "Building WirePod for android/arm (GOARM=7)..."
#GOARCH=arm GOARM=7 GOOS=android $HOME/go/bin/fyne build --os android -o libWirePod-armv7.so -tags nolibopusfile
GOOS=android GOARCH=arm GOARM=7 go build -buildmode=c-shared -o libWirePod-armv7.so -tags nolibopusfile
echo "Putting libraries in vessel APK..."
rm -rf tmp
mkdir -p tmp
cd tmp
cp ../WirePod.apk .
mkdir -p lib/arm64-v8a
mkdir -p lib/armeabi-v7a
cp ../built-libs/arm64/lib/libopus.so lib/arm64-v8a/
cp ../built-libs/arm64/lib/libvosk.so lib/arm64-v8a/
cp ../built-libs/armv7/lib/libopus.so lib/armeabi-v7a/
cp ../built-libs/armv7/lib/libvosk.so lib/armeabi-v7a/
cp ../libWirePod-armv7.so lib/armeabi-v7a/libWirePod.so
cp ../libWirePod-arm64.so lib/arm64-v8a/libWirePod.so
zip -r WirePod.apk lib
${ANDROID_HOME}/build-tools/34.0.0/apksigner sign --ks ../key/ks.jks --ks-pass pass:"$(cat ../key/passwd)" --out ../WirePod.apk WirePod.apk
cd ..
rm -rf tmp
rm -f libWirePod-armv7.so
rm -f libWirePod-arm64.so
rm -f WirePod.apk.idsig
rm -f vessel/WirePod.apk
rm -f resources/static.zip
rm -f static.go
echo "Build complete ./WirePod.apk"
