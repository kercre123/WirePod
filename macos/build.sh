#!/bin/bash

set -e

export PODVER="$1"
export ARCH="$2"

if [[ ${PODVER} == "" ]]; then
	echo "You must provide a version (v1.0.0)."
	exit 0
fi

sudo -u $SUDO_USER brew install autoconf automake libtool create-dmg wget pkg-config

export ORIGDIR="$(pwd)"
export PODLIBS="${ORIGDIR}/libs"

function buildApp() {
    echo
    echo "Building for $1"
    echo

    if [[ $1 == "arm64" ]]; then
        export TARGET="aarch64-apple-darwin"
    elif [[ $1 == "amd64" ]]; then
        export TARGET="x86_64-apple-darwin"
    else
        echo "You must provide a valid architecture (arm64/amd64)."
        exit 1
    fi
    export CC="clang -target ${TARGET}"
    export CXX="clang++ -target ${TARGET}"

    if [[ ! -d ${PODLIBS}/opus/$1 ]]; then
        echo "opus directory doesn't exist. cloning and building"
        mkdir -p ${PODLIBS}/opus/$1
        cd ${PODLIBS}/opus/$1
        git clone https://github.com/xiph/opus . --depth=1
        ./autogen.sh
        ./configure --host=${TARGET} --prefix="${PODLIBS}/opus/$1"
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

    export GOARCH=$1
    export CGO_ENABLED=1
    export CGO_LDFLAGS="-L${PODLIBS}/opus/$1/lib -L${PODLIBS}/vosk -mmacosx-version-min=10.10"
    export CGO_CFLAGS="-I${PODLIBS}/opus/$1/include -I${PODLIBS}/vosk -mmacosx-version-min=10.10"
    export PKG_CONFIG_PATH=$PKG_CONFIG_PATH:${PODLIBS}/opus/$1/lib/pkgconfig
    export SDKROOT="$(xcrun --sdk macosx --show-sdk-path)"

    APPDIR=target/$1/WirePod.app/Contents
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
    -o target/$1/WirePod.app/Contents/MacOS/WirePod \
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
    echo "  <string>$PODVER</string>" >> $PLISTFILE
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

    cp -r ../icons/* ${RESOURCES}
    cp -r ../icons ${RESOURCES}/
    echo "${PODVER}" > ${RESOURCES}/version
    cp ${PODLIBS}/opus/$1/lib/libopus.0.dylib ${FRAMEWORKS}    
    cp ${PODLIBS}/vosk/libvosk.dylib ${FRAMEWORKS}
    cp ${CHPATH}/weather-map.json ${CHIPPER}
    cp -r ${CHPATH}/intent-data ${CHIPPER}
    cp -r ${CHPATH}/webroot ${CHIPPER}
    cp -r ${CHPATH}/epod ${CHIPPER}
    cp ${CLPATH}/build/vic-cloud ${VECTOR_CLOUD}/build/
    cp ${CLPATH}/pod-bot-install.sh ${VECTOR_CLOUD}

    sudo install_name_tool \
    -change ${PODLIBS}/opus/$1/lib/libopus.0.dylib \
    @executable_path/../Frameworks/libopus.0.dylib \
    ${APPDIR}/MacOS/WirePod

    sudo install_name_tool \
    -change libvosk.dylib \
    @executable_path/../Frameworks/libvosk.dylib \
    ${APPDIR}/MacOS/WirePod
}

function buildDmg() {
    echo
    echo "Creating dmg for $1"
    echo
    sudo create-dmg \
    --volname "WirePod Installer" \
    --window-size 800 450 \
    --icon-size 100 \
    --icon "WirePod.app" 200 200 \
    --hide-extension "WirePod.app" \
    --app-drop-link 600 200 \
    target/$1/WirePod-darwin-$1-${PODVER}.dmg \
    target/$1/
}

rm -rf target
if [[ ${ARCH} == "" ]]; then
    echo "No architecture specified. Building for both arm64 and amd64."
    buildApp "arm64"
    buildApp "amd64"
    buildDmg "arm64"
    buildDmg "amd64"
else
    buildApp ${ARCH}
    buildDmg ${ARCH}
fi