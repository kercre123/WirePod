#!/bin/bash

export BUILDFILES="./installer"

export ARCHITECTURE=amd64

set -e

export PODVER="$1"

if [[ ${PODVER} == "" ]]; then
	echo "You must provide a version (v1.0.0)."
	exit 0
fi

if [[ ! -f .aptDone ]]; then
    sudo apt install mingw-w64 zip build-essential autoconf unzip upx
    touch .aptDone
fi

export GOOS=windows
export GOARCH=amd64
export ARCHITECTURE=amd64
export CGO_ENABLED=1
if [[ "$(uname -s)" == "Darwin" ]]; then
    export CC=x86_64-w64-mingw32-gcc
    export CXX=x86_64-w64-mingw32-g++
    sudo -u $SUDO_USER brew install mingw-w64
else
    export CC=/usr/bin/x86_64-w64-mingw32-gcc
    export CXX=/usr/bin/x86_64-w64-mingw32-g++
fi


x86_64-w64-mingw32-windres installer/rc/app.rc -O coff -o installer/app.syso

go build \
-ldflags "-H=windowsgui -w -s" \
-o WirePodInstaller-${PODVER}.exe \
${BUILDFILES}

#upx windows/wire-pod-installer.exe
