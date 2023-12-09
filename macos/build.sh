#!/bin/bash

set -e

brew install autoconf automake libtool create-dmg go wget pkg-config

export ORIGDIR="$(pwd)"
export PODLIBS="${ORIGDIR}/libs"

export GOOS=darwin
if [[ "$(uname -m)" == "x86_64" ]]; then
    export GOARCH=amd64
elif [[ "$(uname -m)" == "aarch64" ]]; then
    export GOARCH=arm64
else
    echo "Invalid architecture: $(uname -m)"
    exit 1
fi

if [[ ! -d ${PODLIBS}/opus ]]; then
    echo "opus directory doesn't exist. cloning and building"
    mkdir -p ${PODLIBS}
    cd ${PODLIBS}
    git clone https://github.com/xiph/opus --depth=1
    cd opus
    ./autogen.sh
    ./configure --prefix="${PODLIBS}/opus"
    make -j
    make install
    cd ${ORIGDIR}
fi

if [[ ! -d ${PODLIBS}/vosk ]]; then
    echo "getting vosk from alphacep releases page"
    cd ${PODLIBS}
    wget https://github.com/alphacep/vosk-api/releases/download/v0.3.42/vosk-osx-0.3.42.zip
    unzip vosk-osx-0.3.42.zip
    mv vosk-osx-0.3.42 vosk
    cd ${ORIGDIR}
fi

export CGO_ENABLED=1
export CGO_LDFLAGS="-L${PODLIBS}/opus/lib -L${PODLIBS}/vosk"
export CGO_CFLAGS="-I${PODLIBS}/opus/include -I${PODLIBS}/vosk"
export PKG_CONFIG_PATH=$PKG_CONFIG_PATH:${PODLIBS}/opus/lib/pkgconfig

rm -rf target

APPDIR=target/app/WirePod.app/Contents
PLISTFILE=${APPDIR}/Info.plist
RESOURCES=${APPDIR}/Resources
FRAMEWORKS=${APPDIR}/Frameworks
CHIPPER=${APPDIR}/Frameworks/chipper
VECTOR_CLOUD=${APPDIR}/Frameworks/vector-cloud

mkdir -p ${RESOURCES}
mkdir -p ${FRAMEWORKS}
mkdir -p ${CHIPPER}
mkdir -p ${VECTOR_CLOUD}/build

go build \
-tags nolibopusfile \
-o target/app/WirePod.app/Contents/MacOS/WirePod \
./cmd

echo "<?xml version="1.0" encoding="UTF-8"?>" > $PLISTFILE
echo "<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">" >> $PLISTFILE
echo "<plist version="1.0">" >> $PLISTFILE
echo "<dict>" >> $PLISTFILE
echo "  <key>CFBundleGetInfoString</key>" >> $PLISTFILE
echo "  <string>WirePod</string>" >> $PLISTFILE
echo "  <key>CFBundleExecutable</key>" >> $PLISTFILE
echo "  <string>WirePod</string>" >> $PLISTFILE
echo "  <key>CFBundleIdentifier</key>" >> $PLISTFILE
echo "  <string>io.github.kercre123</string>" >> $PLISTFILE
echo "  <key>CFBundleName</key>" >> $PLISTFILE
echo "  <string>WirePod</string>" >> $PLISTFILE
echo "  <key>CFBundleIconFile</key>" >> $PLISTFILE
echo "  <string>icon.icns</string>" >> $PLISTFILE
echo "  <key>CFBundleVersion</key>" >> $PLISTFILE
echo "  <string>0.0.1</string>" >> $PLISTFILE
echo "  <key>CFBundleInfoDictionaryVersion</key>" >> $PLISTFILE
echo "  <string>6.0</string>" >> $PLISTFILE
echo "  <key>CFBundlePackageType</key>" >> $PLISTFILE
echo "  <string>APPL</string>" >> $PLISTFILE
echo "  <key>NSHighResolutionCapable</key><true/>" >> $PLISTFILE
echo "  <key>NSSupportsAutomaticGraphicsSwitching</key><true/>" >> $PLISTFILE
echo "  <key>LSUIElement</key><true/>" >> $PLISTFILE
echo "</dict>" >> $PLISTFILE
echo "</plist>" >> $PLISTFILE

export CHPATH="../wire-pod/chipper"
export CLPATH="../wire-pod/vector-cloud"
cp -r ../icons/ ${RESOURCES}
cp ${PODLIBS}/opus/lib/libopus.0.dylib ${FRAMEWORKS}    
cp ${PODLIBS}/vosk/libvosk.dylib ${FRAMEWORKS}
cp ${CHPATH}/weather-map.json ${CHIPPER}
cp -r ${CHPATH}/intent-data ${CHIPPER}
cp -r ${CHPATH}/webroot ${CHIPPER}
cp -r ${CHPATH}/epod ${CHIPPER}
cp ${CLPATH}/build/vic-cloud ${VECTOR_CLOUD}/build/
cp ${CLPATH}/pod-bot-install.sh ${VECTOR_CLOUD}

sudo install_name_tool \
-change ${PODLIBS}/opus/lib/libopus.0.dylib \
@executable_path/../Frameworks/libopus.0.dylib \
${APPDIR}/MacOS/WirePod

sudo install_name_tool \
-change libvosk.dylib \
@executable_path/../Frameworks/libvosk.dylib \
${APPDIR}/MacOS/WirePod

mkdir target/installer
sudo create-dmg \
--volname "WirePod Installer" \
--window-size 800 450 \
--icon-size 100 \
--icon "WirePod.app" 200 200 \
--hide-extension "WirePod.app" \
--app-drop-link 600 200 \
--hdiutil-quiet \
target/installer/WirePod-${GOOS}-${GOARCH}.dmg \
target/app/
