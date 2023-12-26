package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	cross_win "github.com/kercre123/WirePod/cross/win"
	"github.com/ncruces/zenity"
)

var discrete bool

func StopWirePodIfRunning() {
	podPid, err := os.ReadFile(filepath.Join(os.TempDir(), "/wirepodrunningPID"))
	if err == nil {
		pid, _ := strconv.Atoi(string(podPid))
		// doesn't work on unix, but should on Windows
		isRunning, err := cross_win.IsProcessRunning(pid)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if isRunning {
			podProcess, err := os.FindProcess(pid)
			if err == nil {
				fmt.Println("Stopping wire-pod")
				podProcess.Kill()
				podProcess.Wait()
				fmt.Println("Stopped")
			}
		}
	}
	CheckWirePodRunningViaRegistry()
}

func CheckWirePodRunningViaRegistry() {
	pid, err := cross_win.GetRegistryValueInt(cross_win.SoftwareKey, "LastRunningPID")
	if err != nil {
		return
	}
	isRunning, err := cross_win.IsProcessRunning(pid)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if isRunning {
		podProcess, err := os.FindProcess(pid)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Stopping wire-pod")
		podProcess.Kill()
		podProcess.Wait()
		fmt.Println("Stopped")
	}
}

func main() {
	cross_win.InitReg()
	if os.Getenv("RUN_DISCRETE") == "true" {
		discrete = true
	}
	if !discrete {
		err := zenity.Question(
			"Are you sure you want to uninstall wire-pod?",
			zenity.QuestionIcon,
			zenity.Title("wire-pod uninstaller"),
			zenity.OKLabel("Yes"),
		)
		if errors.Is(err, zenity.ErrCanceled) {
			os.Exit(1)
		}
	}
	StopWirePodIfRunning()
	if !discrete {
		err := zenity.Question(
			"Would you like to remove application data, including saved bot settings and API preferences?",
			zenity.ExtraButton("No"),
			zenity.QuestionIcon,
			zenity.NoCancel(),
			zenity.Title("wire-pod uninstaller"),
		)
		if err == nil {
			conf, _ := os.UserConfigDir()
			os.RemoveAll(filepath.Join(conf, "wire-pod"))
		}
	}

	val, err := cross_win.GetRegistryValueString(cross_win.SoftwareKey, "InstallPath")

	if err != nil {
		fmt.Println("error getting installpath from registry: " + err.Error())
		os.Exit(0)
	}

	cross_win.DeleteEverythingFromRegistry()
	fmt.Println("Deleted wire-pod from registry")

	fmt.Println(val)

	os.RemoveAll(filepath.Join(val, "chipper"))
	os.RemoveAll(filepath.Join(val, "vector-cloud"))
	os.Remove("C:\\ProgramData\\Microsoft\\Windows\\Start Menu\\Programs\\wire-pod.lnk")
	os.Remove("C:\\ProgramData\\Microsoft\\Windows\\Start Menu\\Programs\\WirePod.lnk")
	if !discrete {
		zenity.Info(
			"wire-pod has successfully been uninstalled.",
			zenity.InfoIcon,
			zenity.Title("wire-pod uninstaller"),
		)
	}
	os.RemoveAll(val)
	os.Exit(0)
}
