#!/bin/bash

export BUILDFILES="./cmd"

WP_COMMIT_HASH=$(cd ../wire-pod && git rev-parse --short HEAD)
GOLDFLAGS="-X 'github.com/kercre123/wire-pod/chipper/pkg/vars.CommitSHA=${WP_COMMIT_HASH}'"

if [[ "$(uname -s)" == "Darwin" ]]; then
    export CC=x86_64-w64-mingw32-gcc
    export CXX=x86_64-w64-mingw32-g++
else
    export CC=/usr/bin/x86_64-w64-mingw32-gcc
    export CXX=/usr/bin/x86_64-w64-mingw32-g++
fi
export PODHOST=x86_64-w64-mingw32
export ARCHITECTURE=amd64
export CHPATH="../wire-pod/chipper"
export CLPATH="../wire-pod/vector-cloud"

set -e

# if [[ ! -f .aptDone ]]; then
#     sudo apt install mingw-w64 zip build-essential autoconf unzip
#     touch .aptDone
# fi

export ORIGDIR="$(pwd)"
export PODLIBS="${ORIGDIR}/libs"

mkdir -p "${PODLIBS}"

if [[ ! -d "${PODLIBS}/ogg" ]]; then
    echo "ogg directory doesn't exist. cloning and building"
    rm -rf ogg
    git clone https://github.com/xiph/ogg --depth=1
    cd ogg
    ./autogen.sh
    ./configure --host=${PODHOST} --prefix="${PODLIBS}/ogg"
    make -j
    make install
    cd "${ORIGDIR}"
fi

if [[ ! -d "${PODLIBS}/opus" ]]; then
    echo "opus directory doesn't exist. cloning and building"
    rm -rf opus
    git clone https://github.com/xiph/opus --depth=1
    cd opus
    ./autogen.sh
    ./configure --host=${PODHOST} --prefix="${PODLIBS}/opus"
    make -j
    make install
    cd "${ORIGDIR}"
fi

if [[ ! -d ${PODLIBS}/vosk ]]; then
    echo "getting vosk from alphacep releases page"
    cd "${PODLIBS}"
    wget https://github.com/alphacep/vosk-api/releases/download/v0.3.45/vosk-win64-0.3.45.zip
    unzip vosk-win64-0.3.45.zip
    mv vosk-win64-0.3.45 vosk
    cd "${ORIGDIR}"
fi

export GOOS=windows
export GOARCH=amd64
export ARCHITECTURE=amd64
export GO_TAGS="nolibopusfile"

export CGO_ENABLED=1
export CGO_LDFLAGS="-L${PODLIBS}/ogg/lib -L${PODLIBS}/opus/lib -L${PODLIBS}/vosk"
export CGO_CFLAGS="-I${PODLIBS}/ogg/include -I${PODLIBS}/opus/include -I${PODLIBS}/vosk"
export PKG_CONFIG_PATH=$PKG_CONFIG_PATH:${PODLIBS}/opus/lib/pkgconfig

x86_64-w64-mingw32-windres cmd/rc/app.rc -O coff -o cmd/app.syso

go build \
-tags ${GO_TAGS} \
-ldflags "-H=windowsgui -w -s ${GOLDFLAGS}" \
-o chipper.exe \
${BUILDFILES}

go build \
-tags ${GO_TAGS} \
-ldflags "-H=windowsgui -w -s" \
-o uninstall.exe \
./uninstall/main.go

rm -rf tmp
mkdir -p tmp/wire-pod/chipper
mkdir -p tmp/wire-pod/vector-cloud/build

cp -r ${CHPATH}/intent-data tmp/wire-pod/chipper/
cp ${CHPATH}/weather-map.json tmp/wire-pod/chipper/
cp -r ${CHPATH}/webroot tmp/wire-pod/chipper/
cp -r ${CHPATH}/epod tmp/wire-pod/chipper/
cp ${CHPATH}/stttest.pcm tmp/wire-pod/chipper/
echo $1 > tmp/wire-pod/chipper/version
cp ${CLPATH}/build/vic-cloud tmp/wire-pod/vector-cloud/build/
cp ${CLPATH}/pod-bot-install.sh tmp/wire-pod/vector-cloud/
cp -r ../icons tmp/wire-pod/chipper/icons

# echo "export DEBUG_LOGGING=true" > tmp/botpack/wire-pod/chipper/source.sh
# echo "export STT_SERVICE=vosk" >> tmp/botpack/wire-pod/chipper/source.sh

cp uninstall.exe tmp/wire-pod/
cp chipper.exe tmp/wire-pod/chipper/

cp ${PODLIBS}/opus/bin/libopus-0.dll tmp/wire-pod/chipper/
cp ${PODLIBS}/ogg/bin/libogg-0.dll tmp/wire-pod/chipper/
cp ${PODLIBS}/vosk/* tmp/wire-pod/chipper/
rm tmp/wire-pod/chipper/libvosk.lib

cd tmp

rm -rf ../wire-pod-win-${ARCHITECTURE}.zip

zip -r ../wire-pod-win-${ARCHITECTURE}.zip wire-pod

cd ..
rm -rf tmp
rm chipper.exe
rm uninstall.exe
