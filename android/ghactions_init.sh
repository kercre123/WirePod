#!/bin/bash

KEY=$1
PASSWD=$2

echo $KEY | base64 -d > key/ks.jks
echo $PASSWD > key/passwd

wget https://dl.google.com/android/repository/android-ndk-r23c-linux.zip
wget https://mirrors.cloud.tencent.com/AndroidSDK/build-tools_r34-linux.zip

unzip android-ndk-r23c-linux.zip
unzip build-tools_r34-linux.zip

rm android-ndk-r23c-linux.zip
rm build-tools_r34-linux.zip