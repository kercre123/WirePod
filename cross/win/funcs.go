package cross_win

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	all "github.com/kercre123/WirePod/cross/all"
)

type Windows struct {
	all.OSFuncs
}

func NewWindows() *Windows {
	var WindowsObj *Windows
	return WindowsObj
}

func (w *Windows) Init() error {
	return InitReg()
}

func (w *Windows) ReadConfig() (all.WPConfig, error) {
	var wp all.WPConfig
	port, err := GetRegistryValueString(SoftwareKey, "WebPort")
	if err != nil {
		return wp, err
	}
	ver, _ := GetRegistryValueString(SoftwareKey, "PodVersion")
	path, _ := GetRegistryValueString(SoftwareKey, "InstallPath")
	runatstartup, _ := GetRegistryValueString(SoftwareKey, "RunAtStartup")
	needsr, _ := GetRegistryValueString(SoftwareKey, "NeedsRestart")
	pid, _ := GetRegistryValueInt(SoftwareKey, "LastRunningPID")
	wp.WSPort = port
	wp.Version = ver
	wp.InstallPath = path
	wp.LastRunningPID = pid
	if runatstartup == "true" {
		wp.RunAtStartup = true
	}
	if needsr == "true" {
		wp.NeedsRestart = true
	}
	return wp, nil
}

func (w *Windows) WriteConfig(wp all.WPConfig) error {
	err := UpdateRegistryValueString(SoftwareKey, "InstallPath", wp.InstallPath)
	if err != nil {
		return err
	}
	UpdateRegistryValueString(SoftwareKey, "PodVersion", wp.Version)
	UpdateRegistryValueString(SoftwareKey, "WebPort", wp.WSPort)
	UpdateRegistryValueString(SoftwareKey, "RunAtStartup", fmt.Sprint(wp.RunAtStartup))
	UpdateRegistryValueString(SoftwareKey, "NeedsRestart", fmt.Sprint(wp.NeedsRestart))
	UpdateRegistryValueInt(SoftwareKey, "LastRunningPID", wp.LastRunningPID)
	return nil
}

func (w *Windows) RunPodAtStartup(run bool) error {
	conf, err := w.ReadConfig()
	if err != nil {
		return err
	}
	if run {
		cmd := fmt.Sprintf(`cmd.exe /C start "" "` + filepath.Join(conf.InstallPath, "chipper\\chipper.exe") + `" -d`)
		UpdateRegistryValueString(StartupRunKey, "wire-pod", cmd)
		conf.RunAtStartup = true
	} else {
		DeleteRegistryValue(StartupRunKey, "wire-pod")
		conf.RunAtStartup = false
	}
	err = w.WriteConfig(conf)
	if err != nil {
		return err
	}
	return nil
}

func (w *Windows) IsPodAlreadyRunning() bool {
	conf, err := w.ReadConfig()
	if err != nil {
		return false
	}
	if conf.LastRunningPID == 0 {
		return false
	}
	isRunning, err := IsProcessRunning(conf.LastRunningPID)
	if err != nil {
		fmt.Println("syscall error")
		panic(err)
	}
	return isRunning
}

func (w *Windows) IsPIDProcessRunning(pid int) (bool, error) {
	return IsProcessRunning(pid)
}

func (w *Windows) KillExistingPod() error {
	conf, err := w.ReadConfig()
	if err != nil {
		return err
	}
	if conf.LastRunningPID == 0 {
		return errors.New("no pod running (pid: 0)")
	}
	proc, err := os.FindProcess(conf.LastRunningPID)
	if err != nil {
		return err
	}
	proc.Kill()
	return nil
}

func (w *Windows) OnExit() {
	conf, err := w.ReadConfig()
	if err != nil {
		return
	}
	conf.LastRunningPID = 0
	w.WriteConfig(conf)
}
