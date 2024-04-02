package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	botsetup "github.com/kercre123/wire-pod/chipper/pkg/wirepod/setup"
	stt "github.com/kercre123/wire-pod/chipper/pkg/wirepod/stt/vosk"
	"gopkg.in/ini.v1"
)

func main() {
	vars.IsPackagedLinux = true
	verb := flag.Bool("verbose", true, "with/without debug logging")
	justIP := flag.Bool("justip", false, "show just configuration page")
	flag.Parse()
	var webPort string
	var useVoskGrammer bool
	f, err := ini.Load("/etc/wire-pod/config.ini")
	if err != nil {
		fmt.Println("Can't find /etc/wire-pod/config.ini, assuming port 8080")
		webPort = "8080"
	} else {
		sec, err := f.GetSection("")
		if err != nil {
			fmt.Println("Error reading INI, assuming port 8080")
			webPort = "8080"
		} else {
			k, err := sec.GetKey("web_port")
			if err != nil {
				fmt.Println("Error reading INI, assuming port 8080")
				webPort = "8080"
			} else {
				webPort = strings.TrimSpace(k.String())
			}
			k, err = sec.GetKey("vosk_with_grammer")
			if err != nil {
				useVoskGrammer = false
			} else {
				if strings.TrimSpace(k.String()) == "true" {
					useVoskGrammer = true
				}
			}
		}
	}
	if *justIP {
		ipAddr := botsetup.GetOutboundIP().String()
		fmt.Println("\033[1;32mWirePod configuration page: \033[1;36mhttp://" + ipAddr + ":" + webPort + "\033[0m")
		os.Exit(0)
	}
	os.Setenv("WEBSERVER_PORT", webPort)
	if useVoskGrammer {
		os.Setenv("VOSK_WITH_GRAMMER", "true")
	}
	vars.Packaged = true
	os.UserConfigDir()
	_, err = os.Open("/etc/wire-pod")
	if err != nil {
		fmt.Println("FATAL: no /etc/wire-pod folder exists :(")
		os.Exit(1)
	}
	if *verb {
		os.Setenv("DEBUG_LOGGING", "true")
	}
	os.Setenv("STT_SERVICE", "vosk")
	os.Chdir("/etc/wire-pod")
	StartFromProgramInit(stt.Init, stt.STT, stt.Name)
}
