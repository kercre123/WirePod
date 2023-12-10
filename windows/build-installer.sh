#!/bin/bash

export BUILDFILES="./installer"

export ARCHITECTURE=amd64

set -e

if [[ ! -f .aptDone ]]; then
    sudo apt install mingw-w64 zip build-essential autoconf unzip upx
    touch .aptDone
fi

export GOOS=windows
export ARCHITECTURE=amd64
export CGO_ENABLED=1
if [[ "$(uname -s)" == "Darwin" ]]; then
    export CC=/usr/local/bin/x86_64-w64-mingw32-gcc
    export CXX=/usr/local/bin/x86_64-w64-mingw32-g++
else
    export CC=/usr/bin/x86_64-w64-mingw32-gcc
    export CXX=/usr/bin/x86_64-w64-mingw32-g++
fi


x86_64-w64-mingw32-windres installer/rc/app.rc -O coff -o installer/app.syso

go build \
-ldflags "-H=windowsgui" \
-o wire-pod-installer.exe \
${BUILDFILES}

#upx windows/wire-pod-installer.exe
