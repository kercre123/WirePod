#!/bin/bash

echo "$ANDROID_KEY" | base64 -d > key/ks.jks
echo "$ANDROID_PASSWD" > key/passwd

if [[ ! -d android-ndk ]]; then
    echo "Getting ndk..."
    wget -q https://dl.google.com/android/repository/android-ndk-r23c-linux.zip
    echo "Unzipping ndk..."
    unzip -qq android-ndk-r23c-linux.zip
    rm android-ndk-r23c-linux.zip
    mkdir android-ndk
    mv android-ndk-r23c android-ndk/ndk-bundle
fi

if [[ ! -d android-14 ]]; then
    echo "Getting build-tools..."
    wget -q https://mirrors.cloud.tencent.com/AndroidSDK/build-tools_r34-linux.zip
    echo "Unzipping build-tools..."
    unzip -qq build-tools_r34-linux.zip
    rm build-tools_r34-linux.zip
fi