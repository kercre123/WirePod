package main

import (
	"os"

	"github.com/kercre123/wire-pod/chipper/pkg/initwirepod"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	stt "github.com/kercre123/wire-pod/chipper/pkg/wirepod/stt/vosk"
)

func main() {
	vars.Packaged = true
	_, err := os.Open("/etc/wire-pod")
	if err != nil {

	}
	os.Setenv("DEBUG_LOGGING", "true")
	os.Setenv("STT_SERVICE", "vosk")
	os.Chdir("/etc/wire-pod")
	initwirepod.StartFromProgramInit(stt.Init, stt.STT, stt.Name)
}
