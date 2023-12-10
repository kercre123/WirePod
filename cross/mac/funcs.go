package cross_mac

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	all "github.com/kercre123/WirePod/cross/all"
	"github.com/ncruces/zenity"
)

type MacOS struct {
	all.OSFuncs
}

func NewMacOS() *MacOS {
	var obj *MacOS
	return obj
}

func RunSudoCommand(cmd string) error {
	return exec.Command("osascript", "-e", fmt.Sprintf("do shell script \"%s\" with administrator privileges", cmd)).Run()
}

func (w *MacOS) Init() error {
	execu, _ := os.Executable()
	if !strings.HasPrefix(execu, "/Applications/") {
		zenity.Error(
			"WirePod must be copied to the Applications folder before execution.",
			zenity.ErrorIcon,
			zenity.Title("WirePod error"),
			zenity.OKLabel("Quit WirePod"),
		)
		os.Exit(0)
	}

	conf, _ := w.ReadConfig()
	if conf.FirstStartup {
		err := zenity.Info(
			"Would you like WirePod to run when the user logs in?",
			zenity.Title("WirePod"),
			zenity.OKLabel("Yes"),
			zenity.ExtraButton("No"),
			zenity.QuestionIcon,
		)
		if err == nil {
			w.RunPodAtStartup(true)
			conf.RunAtStartup = true
		}
	}
	if conf.FirstStartup && w.Hostname() != "escapepod" {
		err := zenity.Info(
			"Would you like WirePod to set the system's hostname to escapepod? This is required if you want to use a regular, production robot with WirePod. This will require a computer restart.",
			zenity.Title("WirePod"),
			zenity.OKLabel("Yes"),
			zenity.ExtraButton("No"),
			zenity.QuestionIcon,
		)
		if err != zenity.ErrExtraButton {
			conf.FirstStartup = false
			w.WriteConfig(conf)
			RunSudoCommand("scutil --set LocalHostName escapepod")
			err = zenity.Info(
				"The hostname has been set! Your Mac must now be restarted before you start WirePod. (Restart Later will exit WirePod)",
				zenity.InfoIcon,
				zenity.Title("WirePod"),
				zenity.ExtraButton("Restart Later"),
				zenity.OKLabel("Restart Now"),
			)
			if err == zenity.ErrExtraButton {
				os.Exit(0)
			} else {
				RunSudoCommand("shutdown -r now")
			}
		}
	}
	if conf.FirstStartup {
		conf, _ = w.ReadConfig()
		conf.FirstStartup = false
		w.WriteConfig(conf)
	}
	return nil
}

func MakeDefaultConfig() all.WPConfig {
	var conf all.WPConfig
	execu, _ := os.Executable()
	conf.InstallPath = filepath.Dir(execu) + "/../Frameworks"
	conf.RunAtStartup = false
	conf.NeedsRestart = false
	ver, err := os.ReadFile(filepath.Join(filepath.Dir(execu), "/../Resources/version"))
	if err != nil {
		conf.Version = "v0.0.1"
	} else {
		conf.Version = strings.TrimSpace(string(ver))
	}
	conf.FirstStartup = true
	conf.WSPort = "8080"
	return conf
}

func (w *MacOS) ReadConfig() (all.WPConfig, error) {
	coDir, _ := os.UserConfigDir()
	confFile := filepath.Join(coDir, "wire-pod") + "/wire-pod-conf.json"
	file, err := os.ReadFile(confFile)
	if err != nil {
		conf := MakeDefaultConfig()
		w.WriteConfig(conf)
		return conf, nil
	}
	var conf all.WPConfig
	json.Unmarshal(file, &conf)
	return conf, nil
}

func (w *MacOS) WriteConfig(conf all.WPConfig) error {
	coDir, _ := os.UserConfigDir()
	os.MkdirAll(filepath.Join(coDir, "wire-pod"), 0777)
	confFile := filepath.Join(coDir, "wire-pod") + "/wire-pod-conf.json"
	marshalled, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	os.WriteFile(confFile, marshalled, 0777)
	return nil
}

func (w *MacOS) Hostname() string {
	output, _ := exec.Command("scutil", "--get", "LocalHostName").Output()
	hostname := strings.TrimSuffix(string(output[:]), "\n")
	return hostname
}

func (w *MacOS) ResourcesPath() string {
	appPath, _ := os.Executable()
	return filepath.Dir(appPath) + "/../Resources/"
}

// macOS handles this, we don't need to handle it

func (w *MacOS) IsPIDProcessRunning(pid int) (bool, error) {
	// if pid == 0 {
	// 	return false, nil
	// }
	// process, err := os.FindProcess(pid)
	// if err != nil {
	// 	return false, nil
	// }
	// err = process.Signal(syscall.Signal(0))
	// return err == nil, nil
	return false, nil
}

func (w *MacOS) IsPodAlreadyRunning() bool {
	conf, _ := w.ReadConfig()
	isRunning, _ := w.IsPIDProcessRunning(conf.LastRunningPID)
	return isRunning
}

// don't need to implement as we don't have an installer
func (w *MacOS) KillExistingPod() error {
	return nil
}

func (w *MacOS) OnExit() {
	conf, _ := w.ReadConfig()
	conf.LastRunningPID = 0
	w.WriteConfig(conf)
}

func (w *MacOS) RunPodAtStartup(run bool) error {
	homeDir, _ := os.UserHomeDir()
	if run {
		launchAgentsDir := filepath.Join(homeDir, "/Library/LaunchAgents")
		if _, err := os.Stat(launchAgentsDir); os.IsNotExist(err) {
			os.Mkdir(launchAgentsDir, 0777)
		}
		executable, _ := os.Executable()
		os.WriteFile(launchAgentsDir+"/WirePod.agent.plist", []byte(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>WirePod.agent</string>
	<key>ProgramArguments</key>
	<array>
		<string>`+executable+`</string>
		<string>-d</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>`), 0777)
	} else {
		os.Remove(filepath.Join(homeDir, "/Library/LaunchAgents/WirePod.agent.plist"))
	}
	conf, _ := w.ReadConfig()
	conf.RunAtStartup = run
	w.WriteConfig(conf)
	return nil
}

func IfFileExist(name string) bool {
	_, err := os.Stat(name)
	return err != nil
}
