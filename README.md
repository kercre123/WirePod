# WirePod

Cross-platform code and resources for [wire-pod](https://github.com/kercre123/wire-pod).

-  wire-pod = actual server code
-  WirePod = code to create user-installable packages of wire-pod

## Support

-  macOS (arm64, amd64)
-  Windows 10/11 (amd64)
-  Android 6.0 and above (kinda)
-  For Linux, use the instructions in wire-pod's wiki.

## Android

-  Right now, WirePod does compile to Android and it does work. There is an APK in the releases page.
-  It does not work with firmware 1.8.1. It only works with 1.7.2.6014ep and 2.0.1.6076ep. Those releases use port 8084 while 1.8.1 uses 443 (which I can't bind to on Android)
-  It is fully featured, but it is still in a proof-of-concept stage.
-  To use:
  -  Install the APK
  -  Go to Android Settings -> Apps -> WirePod and make sure battery optimization is set to Unrestricted or Not restricted
  -  Open the WirePod app
  -  Make sure you are connected to the same Wi-Fi as Vector, and not mobile data
  -  Press Start (under "status"). It should say "running! <url>"
  -  Press "start broadcasting" so your device can be accessible at escapepod.local
  -  Press "start pinging jdocs"
  -  Go to the link under "status" in the phone's browser or another device on the network, and follow the instructions
  -  To setup a bot, use the instructions in the wire-pod wiki. You will probably need to clear user data
