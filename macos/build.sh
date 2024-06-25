#!/bin/bash

set -e

WP_COMMIT_HASH=$(cd ../wire-pod && git rev-parse --short HEAD)
GOLDFLAGS="-X 'github.com/kercre123/wire-pod/chipper/pkg/vars.CommitSHA=${WP_COMMIT_HASH}'"

export PODVER="$1"

if [[ ${PODVER} == "" ]]; then
	echo "You must provide a version (v1.0.0)."
	exit 0
fi

sudo -u $SUDO_USER brew install autoconf automake libtool create-dmg wget pkg-config

export ORIGDIR="$(pwd)"
export PODLIBS="${ORIGDIR}/libs"

function buildBinary() {
    echo
    echo "Building $1 binary"
    echo

    if [[ $1 == "arm64" ]]; then
        export TARGET="aarch64-apple-darwin"
    elif [[ $1 == "amd64" ]]; then
        export TARGET="x86_64-apple-darwin"
    else
        echo "You must provide a valid architecture (arm64/amd64)."
        exit 1
    fi
    export CC="clang -target ${TARGET} -mmacosx-version-min=11"
    export CXX="clang++ -target ${TARGET} -mmacosx-version-min=11"

    if [[ ! -d ${PODLIBS}/opus/$1 ]]; then
        if [[ ! -d opus ]]; then
            echo "opus directory doesn't exist. cloning"
            rm -rf opus
            git clone https://github.com/xiph/opus --depth=1
        fi
        cd opus
        ./autogen.sh
        ./configure --host=${TARGET} --prefix="${PODLIBS}/opus/$1" --disable-doc --disable-extra-programs
        make -j
        make install
        cd ${ORIGDIR}
    fi

    export GOOS=darwin
    export GOARCH=$1
    export CGO_ENABLED=1
    export CGO_LDFLAGS="-L${PODLIBS}/opus/$1/lib -L${PODLIBS}/vosk -mmacosx-version-min=11"
    export CGO_CFLAGS="-I${PODLIBS}/opus/$1/include -I${PODLIBS}/vosk -mmacosx-version-min=11"
    export PKG_CONFIG_PATH=$PKG_CONFIG_PATH:${PODLIBS}/opus/$1/lib/pkgconfig
    export SDKROOT="$(xcrun --sdk macosx --show-sdk-path)"

    go build \
    -tags nolibopusfile \
    -ldflags "-w -s ${GOLDFLAGS}" \
    -o tmp/WirePod-$1 \
    ./cmd
}

function buildApp() {
    echo
    echo "Building app"
    echo

    mkdir -p ${PODLIBS}

    if [[ ! -d ${PODLIBS}/vosk ]]; then
        echo "getting vosk from alphacep releases page"  
        cd ${PODLIBS}
        wget https://github.com/alphacep/vosk-api/releases/download/v0.3.42/vosk-osx-0.3.42.zip
        unzip vosk-osx-0.3.42.zip
        mv vosk-osx-0.3.42 vosk
        cd ${ORIGDIR}
    fi

    APPDIR=target/WirePod.app/Contents
    PLISTFILE=${APPDIR}/Info.plist
    MACOS=${APPDIR}/MacOS
    RESOURCES=${APPDIR}/Resources
    FRAMEWORKS=${APPDIR}/Frameworks
    CHIPPER=${APPDIR}/Frameworks/chipper
    VECTOR_CLOUD=${APPDIR}/Frameworks/vector-cloud

    mkdir -p ${MACOS}
    mkdir -p ${RESOURCES}
    mkdir -p ${FRAMEWORKS}
    mkdir -p ${CHIPPER}
    mkdir -p ${VECTOR_CLOUD}/build

    buildBinary "arm64"
    buildBinary "amd64"

    sudo lipo -create ${PODLIBS}/opus/arm64/lib/libopus.0.dylib ${PODLIBS}/opus/amd64/lib/libopus.0.dylib -output ${PODLIBS}/opus/libopus.0.dylib
    sudo lipo -create tmp/WirePod-arm64 tmp/WirePod-amd64 -output ${MACOS}/WirePod

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
    cp ${PODLIBS}/opus/libopus.0.dylib ${FRAMEWORKS}    
    cp ${PODLIBS}/vosk/libvosk.dylib ${FRAMEWORKS}
    cp ${CHPATH}/weather-map.json ${CHIPPER}
    cp -r ${CHPATH}/intent-data ${CHIPPER}
    cp -r ${CHPATH}/webroot ${CHIPPER}
    cp -r ${CHPATH}/epod ${CHIPPER}
    cp -r ${CHPATH}/stttest.pcm ${CHIPPER}
    echo "${PODVER}" > ${CHIPPER}/version
    cp ${CLPATH}/build/vic-cloud ${VECTOR_CLOUD}/build/
    cp ${CLPATH}/pod-bot-install.sh ${VECTOR_CLOUD}

    sudo install_name_tool \
    -change ${PODLIBS}/opus/arm64/lib/libopus.0.dylib \
    @executable_path/../Frameworks/libopus.0.dylib \
    ${APPDIR}/MacOS/WirePod

    sudo install_name_tool \
    -change ${PODLIBS}/opus/amd64/lib/libopus.0.dylib \
    @executable_path/../Frameworks/libopus.0.dylib \
    ${APPDIR}/MacOS/WirePod

    sudo install_name_tool \
    -change libvosk.dylib \
    @executable_path/../Frameworks/libvosk.dylib \
    ${APPDIR}/MacOS/WirePod
}

function buildDmg() {
    echo
    echo "Creating dmg"
    echo
    sudo create-dmg \
    --volname "WirePod Installer" \
    --window-size 1104 544 \
    --icon-size 100 \
    --icon "WirePod.app" 343 269 \
    --hide-extension "WirePod.app" \
    --background "viceyes.png" \
    --app-drop-link 777 269 \
    target/WirePod-${PODVER}.dmg \
    target/
}

rm -rf target tmp
buildApp
buildDmg
