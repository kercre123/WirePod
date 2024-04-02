#!/bin/bash

#set -e

ORIGPATH="$(pwd)"

AMD64T="$(pwd)/wire-pod-toolchain/x86_64-unknown-linux-gnu/bin/x86_64-unknown-linux-gnu-"
ARMT="$(pwd)/vic-toolchain/arm-linux-gnueabi/bin/arm-linux-gnueabi-"
ARM64T="$(pwd)/wire-pod-toolchain/aarch64-unknown-linux-gnu/bin/aarch64-unknown-linux-gnu-"
#ARM64T="$HOME/x-tools/aarch64-linux-gnu/bin/aarch64-linux-gnu-"

DEBCREATEPATH="$(pwd)/debcreate"

if [[ ! -f vic-toolchain ]]; then
    git clone https://github.com/kercre123/vic-toolchain --depth=1
fi

function prepareVOSKbuild_ARMARM64() {
    cd $ORIGPATH
    ARCH=$1
    if [[ ${ARCH} == "amd64" ]]; then
        echo "prepareVOSKbuild_ARMARM64: this function is for armhf and arm64 only."
        exit 1
    fi
    mkdir -p build/${ARCH}
    mkdir -p built/${ARCH}
    KALDIROOT="$(pwd)/build/${ARCH}/kaldi"
    BPREFIX="$(pwd)/built/${ARCH}"
    cd build/${ARCH}
    expToolchain ${ARCH}
    if [[ ! -f ${KALDIROOT}/KALDIBUILT ]]; then
        git clone -b vosk --single-branch https://github.com/alphacep/kaldi
        cd kaldi/tools
        git clone -b v0.3.20 --single-branch https://github.com/xianyi/OpenBLAS
        git clone -b v3.2.1  --single-branch https://github.com/alphacep/clapack
        echo ${OPENBLAS_ARGS}
        if [[ $ARCH == "armel" ]]; then
            make -C OpenBLAS ONLY_CBLAS=1 TARGET=ARMV7 ${OPENBLAS_ARGS} HOSTCC=/usr/bin/gcc USE_LOCKING=1 USE_THREAD=0 all
        elif [[ $ARCH == "arm64" ]] || [[ $ARCH == "aarch64" ]]; then
            make -C OpenBLAS ONLY_CBLAS=1 TARGET=ARMV8 ${OPENBLAS_ARGS} HOSTCC=/usr/bin/gcc USE_LOCKING=1 USE_THREAD=0 all
        fi
        make -C OpenBLAS ${OPENBLAS_ARGS} HOSTCC=gcc USE_LOCKING=1 USE_THREAD=0 PREFIX=$(pwd)/OpenBLAS/install install
        rm -rf clapack/BUILD
        mkdir -p clapack/BUILD && cd clapack/BUILD
        cmake -DCMAKE_C_FLAGS="$ARCHFLAGS" -DCMAKE_C_COMPILER_TARGET=$PODHOST \
            -DCMAKE_C_COMPILER=$CC -DCMAKE_SYSTEM_NAME=Generic -DCMAKE_AR=$AR \
            -DCMAKE_TRY_COMPILE_TARGET_TYPE=STATIC_LIBRARY \
            -DCMAKE_CROSSCOMPILING=True ..
        make HOSTCC=gcc -j 10 -C F2CLIBS
        make  HOSTCC=gcc -j 10 -C BLAS
        make HOSTCC=gcc  -j 10 -C SRC
        find . -name "*.a" | xargs cp -t ../../OpenBLAS/install/lib
        cd ${KALDIROOT}/tools
        git clone --single-branch https://github.com/alphacep/openfst openfst
        cd openfst
        autoreconf -i
        CFLAGS="-g -O3" ./configure --prefix=${KALDIROOT}/tools/openfst --enable-static --enable-shared --enable-far --enable-ngram-fsts --enable-lookahead-fsts --with-pic --disable-bin --host=${CROSS_TRIPLE} --build=x86-linux-gnu
        make -j 8 && make install
        cd ${KALDIROOT}/src
        sed -i "s:TARGET_ARCH=\"\`uname -m\`\":TARGET_ARCH=$(echo $CROSS_TRIPLE|cut -d - -f 1):g" configure
        sed -i "s: -O1 : -O3 :g" makefiles/linux_openblas_arm.mk
        ./configure --mathlib=OPENBLAS_CLAPACK --shared --use-cuda=no
        make -j 8 online2 lm rnnlm
        find ${KALDIROOT} -name "*.o" -exec rm {} \;
        touch ${KALDIROOT}/KALDIBUILT
    else
        echo "VOSK dependencies already built for $ARCH"
    fi
    cd $ORIGPATH
}

function expToolchain() {
    export CC=${ARMT}gcc
    export CXX=${ARMT}g++
    export LD=${ARMT}ld
    export AR=${ARMT}ar
    export FC=${ARMT}gfortran
    export RANLIB=${ARMT}ranlib
    export AS=${ARMT}as
    export CPP=${ARMT}cpp
    export PODHOST=arm-linux-gnueabi
    export CROSS_TRIPLE=${PODHOST}
    export CROSS_COMPILE=${ARMT}
    export GOARCH=arm
    export GOARM=7
    export GOOS=linux
    export ARCHFLAGS="-mfloat-abi=softfp -mfpu=neon-vfpv4"
}

function doVOSKbuild() {
    ARCH=$1
    cd $ORIGPATH
    KALDIROOT="$(pwd)/build/${ARCH}/kaldi"
    BPREFIX="$(pwd)/built/${ARCH}"
    if [[ ! -f ${BPREFIX}/lib/libvosk.so ]]; then
        cd build/${ARCH}
        expToolchain $ARCH
        if [[ ! -d vosk-api ]]; then
            git clone https://github.com/alphacep/vosk-api --depth=1
        fi
        cd vosk-api/src
        KALDI_ROOT=$KALDIROOT make EXTRA_LDFLAGS="-static-libstdc++" -j8
	cd "${ORIGPATH}/build/${ARCH}"
        mkdir -p "${BPREFIX}/lib"
        mkdir -p "${BPREFIX}/include"
        cp vosk-api/src/libvosk.so "${BPREFIX}/lib/"
        cp vosk-api/src/vosk_api.h "${BPREFIX}/include/"
    else
        echo "VOSK already built for $ARCH"
    fi
    cd $ORIGPATH
}

function buildOPUS() {
    ARCH=$1
    cd $ORIGPATH
    BPREFIX="$(pwd)/built/${ARCH}"
    expToolchain $ARCH
    if [[ ! -f built/${ARCH}/ogg_built ]]; then
        cd build/${ARCH}
        rm -rf ogg
        git clone https://github.com/xiph/ogg --depth=1
        cd ogg
        ./autogen.sh
        ./configure --host=${PODHOST} --prefix=$BPREFIX
        make -j8
        make install
        cd $ORIGPATH
        touch built/${ARCH}/ogg_built
    else
        echo "OGG already built for $ARCH"
    fi

    if [[ ! -f built/${ARCH}/opus_built ]]; then
        cd build/${ARCH}
        rm -rf opus
        git clone https://github.com/xiph/opus --depth=1
        cd opus
        ./autogen.sh
        ./configure --host=${PODHOST} --prefix=$BPREFIX
        make -j8
        make install
        cd $ORIGPATH
        touch built/${ARCH}/opus_built
    else
        echo "OPUS already built for $ARCH"
    fi
}

function buildWirePod() {
    ARCH=$1
    cd $ORIGPATH

    # get the webroot, intent data, certs
    if [[ ! -d wire-pod ]]; then
        git clone https://github.com/kercre123/wire-pod --depth=1
    fi
    DC=debcreate/${ARCH}
    WPC=wire-pod/chipper
    mkdir -p $DC/data/etc/wire-pod
    mkdir -p $DC/data/usr/bin
    mkdir -p $DC/data/usr/lib
    mkdir -p $DC/data/usr/include
    mkdir -p $DC/lib/systemd/system
    mkdir -p debcreate/${ARCH}
    cp -rf $WPC/intent-data $DC/data/etc/wire-pod/
    cp -rf $WPC/epod $DC/data/etc/wire-pod/
    cp -rf $WPC/webroot $DC/data/etc/wire-pod/
    cp -rf $WPC/weather-map.json $DC/data/etc/wire-pod/
    cp -rf wire-pod/vector-cloud/pod-bot-install.sh $DC/data/etc/wire-pod/
    cp -rf built/$ARCH/lib/libvosk.so $DC/data/usr/lib/
    cp -rf built/$ARCH/include/vosk_api.h $DC/data/usr/include/
    cp -rf debfiles/wire-pod.service $DC/lib/systemd/system/
    cp -rf debfiles/config.ini $DC/data/etc/wire-pod/

    # BUILD WIREPOD
    expToolchain $ARCH

    export CGO_ENABLED=1 
    export CGO_LDFLAGS="-L$(pwd)/built/$ARCH/lib -latomic" 
    export CGO_CFLAGS="-I$(pwd)/built/$ARCH/include"

    go build \
    -tags nolibopusfile \
    -ldflags "-w -s" \
    -o $DC/usr/bin/wire-pod \
    ./pod/*.go
}


arch=armel
if [[ ! -f ${ORIGPATH}/built/$arch/lib/libvosk.so ]]; then
    echo "Compiling VOSK dependencies for $arch"
    prepareVOSKbuild_ARMARM64 "$arch"
fi
#    echo "Building VOSK for $arch (if needed)"
doVOSKbuild "$arch"
#    echo "Building OPUS for $arch (if needed)"
buildOPUS "$arch"
echo "Dependencies complete for $arch."
echo "Building wire-pod for $arch..."
#go clean -cache
buildWirePod "$arch"