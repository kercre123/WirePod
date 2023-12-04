package cross_win

import (
	"fmt"
	"path/filepath"
)

// Init() defined in registry.go

type WPConfig struct {
	WSPort       string `json:"wsport"`
	RunAtStartup bool   `json:"runatstartup"`
	InstallPath  string `json:"runtimepath"`
	Version      string `json:"version"`
	// windows-specific
	// if NeedsRestart && hostname != escapepod; then error
	NeedsRestart   bool `json:"needsrestart"`
	LastRunningPID int  `json:"lastrunningpid"`
}

func ReadConfig() (WPConfig, error) {
	var wp WPConfig
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

func WriteConfig(wp WPConfig) error {
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

func RunPodAtStartup(run bool) error {
	conf, err := ReadConfig()
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
	err = WriteConfig(conf)
	if err != nil {
		return err
	}
	return nil
}
