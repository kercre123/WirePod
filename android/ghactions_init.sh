#!/bin/bash

KEY=$1
PASSWD=$2

echo $KEY | base64 -d > key/ks.jks
echo $PASSWD > key/passwd

if [[ ! -d android-ndk ]]; then
    wget https://dl.google.com/android/repository/android-ndk-r23c-linux.zip
    unzip android-ndk-r23c-linux.zip
    rm android-ndk-r23c-linux.zip
    mkdir android-ndk
    mv android-ndk-r23c android-ndk/ndk-bundle
fi

if [[ ! -d android-14 ]]; then
    wget https://mirrors.cloud.tencent.com/AndroidSDK/build-tools_r34-linux.zip
    unzip build-tools_r34-linux.zip
    rm build-tools_r34-linux.zip
fi