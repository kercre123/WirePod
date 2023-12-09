package cross_mac

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	all "github.com/kercre123/WirePod/cross/all"
)

type MacOS struct {
	all.OSFuncs
}

func NewMacOS() *MacOS {
	var obj *MacOS
	return obj
}

func (w *MacOS) Init() error {
	return nil
}

func (w *MacOS) ReadConfig() (all.WPConfig, error) {
	coDir, _ := os.UserConfigDir()
	confFile := filepath.Join(coDir, "wire-pod") + "/wire-pod-conf.json"
	file, err := os.ReadFile(confFile)
	if err != nil {
		return all.WPConfig{}, err
	}
	var conf all.WPConfig
	json.Unmarshal(file, &conf)
	return conf, nil
}

func (w *MacOS) WriteConfig(conf all.WPConfig) error {
	coDir, _ := os.UserConfigDir()
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

// not implementing for now

func (w *MacOS) IsPIDProcessRunning(int) (bool, error) {
	return false, nil
}

func (w *MacOS) IsPodAlreadyRunning() bool {
	return false
}

func (w *MacOS) KillExistingPod() error {
	return nil
}

func (w *MacOS) OnExit() {
}

// end things we need to implement

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
