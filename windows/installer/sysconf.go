package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cross_win "github.com/kercre123/WirePod/cross/win"
)

var GitHubTag string

func UpdateRegistry(is InstallSettings) {
	UpdateUninstallRegistry(is)
	UpdateSoftwareRegistry(is)
}

func DeleteAnyOtherInstallation() {
	instPath, err := cross_win.GetRegistryValueString(cross_win.UninstallKey, "InstallPath")
	if err != nil {
		val, err := cross_win.GetRegistryValueString(cross_win.SoftwareKey, "InstallPath")
		if err != nil {
			return
		}
		fmt.Println("Running uninstaller")
		cmd := exec.Command(val)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "RUN_DISCRETE=true")
		cmd.Run()
		cross_win.DeleteEverythingFromRegistry()
	} else {
		os.RemoveAll(instPath)
		cross_win.DeleteEverythingFromRegistry()
	}
}

func UpdateUninstallRegistry(is InstallSettings) {
	appName := "wire-pod"
	displayIcon := filepath.Join(is.Where, `\chipper\icons\ico\pod256x256.ico`)
	displayVersion := GitHubTag
	publisher := "github.com/kercre123"
	uninstallString := filepath.Join(is.Where, `\uninstall.exe`)
	installLocation := filepath.Join(is.Where, `\chipper\chipper.exe`)
	err := cross_win.UpdateRegistryValueString(cross_win.UninstallKey, "DisplayName", appName)
	if err != nil {
		// if this one works, the rest will
		fmt.Printf("Error setting DisplayName: %v\n", err)
		return
	}
	cross_win.UpdateRegistryValueString(cross_win.UninstallKey, "DisplayIcon", displayIcon)
	cross_win.UpdateRegistryValueString(cross_win.UninstallKey, "DisplayVersion", displayVersion)
	cross_win.UpdateRegistryValueString(cross_win.UninstallKey, "Publisher", publisher)
	cross_win.UpdateRegistryValueString(cross_win.UninstallKey, "UninstallString", uninstallString)
	cross_win.UpdateRegistryValueString(cross_win.UninstallKey, "InstallLocation", installLocation)
	fmt.Println("Registry entries successfully created")
}

func UpdateSoftwareRegistry(is InstallSettings) {
	err := cross_win.UpdateRegistryValueString(cross_win.SoftwareKey, "InstallPath", is.Where)
	if err != nil {
		fmt.Printf("Error setting registry key InstallPath: %v\n", err)
		return
	}
	cross_win.UpdateRegistryValueString(cross_win.SoftwareKey, "PodVersion", GitHubTag)
	cross_win.UpdateRegistryValueString(cross_win.SoftwareKey, "WebPort", is.WebPort)
	cross_win.UpdateRegistryValueString(cross_win.SoftwareKey, "RunAtStartup", fmt.Sprint(is.RunAtStartup))
}

func RunPodAtStartup(is InstallSettings) {
	cmd := fmt.Sprintf(`cmd.exe /C start "" "` + filepath.Join(is.Where, "chipper\\chipper.exe") + `" -d`)
	cross_win.UpdateRegistryValueString(cross_win.StartupRunKey, "wire-pod", cmd)
}

func RebootSystem() error {
	cmd := exec.Command("shutdown", "/r", "/t", "0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reboot system: %v, output: %s", err, output)
	}
	return nil
}

func ChangeHostname(newHostname string) error {
	// Construct the PowerShell command
	psCommand := fmt.Sprintf("Rename-Computer -NewName %s", newHostname)

	// Execute the PowerShell command
	cmd := exec.Command("powershell", "-Command", psCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error executing PowerShell command: %v, output: %s", err, string(output))
	}

	// Output for debugging
	fmt.Println("Command Output:", string(output))

	return nil
}

func UnblockChipper(is InstallSettings) {
	exe := strings.ReplaceAll(filepath.Join(is.Where, "chipper", "chipper.exe"), "'", "''")
	cmd := exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("Unblock-File -LiteralPath '%s'", exe))
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("unblock-file failed: %v, output: %s", err, out)
	}
}

func AllowThroughFirewall(is InstallSettings) {
	exe := filepath.Join(is.Where, "chipper", "chipper.exe")
	cross_win.ClearFirewallRules(exe)
	for _, dir := range []string{"in", "out"} {
		out, err := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
			"name=wire-pod",
			"dir="+dir,
			"action=allow",
			"profile=any",
			"program="+exe,
			"enable=yes").CombinedOutput()
		if err != nil {
			log.Printf("failed to add %s firewall rule: %v, output: %s", dir, err, out)
			continue
		}
		log.Printf("added %s firewall rule", dir)
	}
}

func StopWirePod_Registry() {
	val, err := cross_win.GetRegistryValueInt(cross_win.SoftwareKey, "LastRunningPID")
	if err != nil {
		fmt.Println("wire-pod is not running (good): " + err.Error())
		return
	}

	isRunning, err := cross_win.IsProcessRunning(val)
	if err != nil {
		fmt.Println("Error seeing if wire-pod is running (isprocessrunning): " + err.Error())
		return
	}
	if isRunning {
		podProcess, err := os.FindProcess(val)
		if err != nil {
			fmt.Println("Error seeing if wire-pod is running (findprocess): " + err.Error())
			return
		}
		fmt.Println("Stopping wire-pod")
		podProcess.Kill()
		podProcess.Wait()
		fmt.Println("Stopped")
	}
}
