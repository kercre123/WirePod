package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	stt "github.com/kercre123/wire-pod/chipper/pkg/wirepod/stt/vosk"
	"gopkg.in/ini.v1"
)

func CheckIfPi() bool {
	model, err := os.ReadFile("/proc/device-tree/model")
	if err != nil {
		fmt.Println("Error getting device model (/proc/device-tree/model)")
		return false
	}
	if strings.Contains(string(model), "Raspberry Pi") {
		return true
	}
	return false
}

func PerformanceGov() {
	for i := 0; i < runtime.NumCPU(); i++ {
		fmt.Println("Setting CPU " + fmt.Sprint(i) + " to performance")
		exec.Command("/bin/bash", "-c", "echo performance > /sys/devices/system/cpu/cpu"+fmt.Sprint(i)+"/cpufreq/scaling_governor").Run()
	}
}

func DoPerfMode(perfMode string) {
	switch mode := perfMode; mode {
	case "true":
		PerformanceGov()
	case "onlyifpi":
		if CheckIfPi() {
			PerformanceGov()
		}
	}
}

func main() {
	vars.IsPackagedLinux = true
	verb := flag.Bool("verbose", true, "with/without debug logging")
	justIP := flag.Bool("justip", false, "show just configuration page")
	flag.Parse()
	var webPort string
	var perfMode string
	var useVoskGrammer bool
	var useMdns bool
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
			k, err = sec.GetKey("perf_mode")
			if err != nil {
				perfMode = "onlyifpi"
			} else {
				perfMode = strings.TrimSpace(k.String())
				if perfMode == "" {
					perfMode = "onlyifpi"
				}
			}
			k, err = sec.GetKey("vosk_with_grammer")
			if err != nil {
				useVoskGrammer = false
			} else {
				if strings.TrimSpace(k.String()) == "true" {
					useVoskGrammer = true
				}
			}
			k, err = sec.GetKey("use_mdns")
			if err != nil {
				useMdns = true
			} else {
				if strings.TrimSpace(k.String()) == "true" {
					useMdns = true
				}
			}
		}
	}
	if *justIP {
		ipAddr := vars.GetOutboundIP().String()
		fmt.Println("\033[1;32mWirePod configuration page: \033[1;36mhttp://" + ipAddr + ":" + webPort + "\033[0m")
		os.Exit(0)
	}
	os.Setenv("WEBSERVER_PORT", webPort)
	if useVoskGrammer {
		os.Setenv("VOSK_WITH_GRAMMER", "true")
	}
	if !useMdns {
		os.Setenv("DISABLE_MDNS", "true")
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
	DoPerfMode(perfMode)
	StartFromProgramInit(stt.Init, stt.STT, stt.Name)
}
