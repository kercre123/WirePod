package podapp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"

	"github.com/getlantern/systray"
	all "github.com/kercre123/WirePod/cross/all"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	stt "github.com/kercre123/wire-pod/chipper/pkg/wirepod/stt/vosk"
	"github.com/ncruces/zenity"
)

// this directory contains code which compiled a single program for end users. gui elements are implemented.

var cross all.OSFuncs

var InstallPath string

var mBoxTitle = "WirePod"
var mBoxError = `There was an error starting WirePod: `
var mBoxAlreadyRunning = "WirePod is already running. You must quit that instance before starting another one. Exiting."
var mBoxSuccess = `WirePod has started successfully! It is now running in the background and can be managed in the system tray.`

func mBoxIcon() string {
	return filepath.Join(cross.ResourcesPath(), "icons/png/"+"podfull.png")
}

func getNeedsSetupMsg() string {
	return `WirePod is now running in the background. You must set it up by heading to http://` + vars.GetOutboundIP().String() + `:` + vars.WebPort + ` in a browser.`
}

func checkIfRestartNeeded() bool {
	host, _ := os.Hostname()
	conf, err := cross.ReadConfig()
	if err != nil {
		return false
	}
	if conf.NeedsRestart && host != "escapepod" {
		return true
	} else if conf.NeedsRestart && host == "escapepod" {
		conf.NeedsRestart = false
		cross.WriteConfig(conf)
		return false
	}
	return false
}

// the actual entrypoint function!
func StartWirePod(crossOS all.OSFuncs) {
	cross = crossOS

	defer func() {
		if r := recover(); r != nil {
			conf, _ := os.UserConfigDir()
			dumpFile := filepath.Join(conf, "wire-pod", "dump.txt")
			os.MkdirAll(filepath.Join(conf, "wire-pod"), 0777)
			os.WriteFile(dumpFile, []byte(fmt.Sprint(r)+"\n\n\n"+string(debug.Stack())), 0777)
			fmt.Printf("panic!: %v\n", r)
			zenity.Error("wire-pod has crashed. dump located in "+dumpFile+". exiting",
				zenity.ErrorIcon,
				zenity.Title("wire-pod crash :("))
			ExitProgram(1)
		}
	}()

	err := cross.Init()
	if err != nil {
		ErrMsg(err)
	}
	if checkIfRestartNeeded() {
		zenity.Error(
			"You must restart your computer before starting WirePod.",
			zenity.ErrorIcon,
			zenity.Title(mBoxTitle),
		)
		os.Exit(1)
	}
	vars.Packaged = true
	confDir, err := os.UserConfigDir()
	if err != nil {
		ErrMsg(err)
	}
	pidFile, err := os.ReadFile(confDir + "/runningPID")
	if err == nil {
		pid, _ := strconv.Atoi(string(pidFile))
		if is, _ := cross.IsPIDProcessRunning(pid); is {
			zenity.Error(
				"WirePod is already running.",
				zenity.ErrorIcon,
				zenity.Title(mBoxTitle),
			)
			os.Exit(1)
		}
	}
	if cross.IsPodAlreadyRunning() {
		zenity.Error(
			"WirePod is already running.",
			zenity.ErrorIcon,
			zenity.Title(mBoxTitle),
		)
		os.Exit(1)
	}

	conf, err := cross.ReadConfig()
	if err != nil {
		ErrMsg(err)
	}
	conf.LastRunningPID = os.Getpid()
	err = cross.WriteConfig(conf)
	if err != nil {
		ErrMsg(err)
	}

	err = os.Chdir(filepath.Join(conf.InstallPath, "chipper"))
	fmt.Println("Working directory: " + conf.InstallPath + "/chipper")
	if err != nil {
		ErrMsg(fmt.Errorf("error setting runtime directory to " + conf.InstallPath + "/chipper"))
	}

	if conf.WSPort != "8080" && conf.WSPort != "0" {
		os.Setenv("WEBSERVER_PORT", conf.WSPort)
	}

	systray.Run(onReady, onExit)
}

func ExitProgram(code int) {
	cross.OnExit()
	systray.Quit()
	os.Exit(code)
}

func onExit() {
	os.Exit(0)
}

func onReady() {
	// windows-specific

	os.Setenv("STT_SERVICE", "vosk")
	os.Setenv("DEBUG_LOGGING", "true")

	systrayIcon, err := os.ReadFile(filepath.Join(cross.ResourcesPath(), "icons/ico") + "/pod24x24.ico")
	if err != nil {
		zenity.Error(
			"Error, could not load systray icon. Something is wrong with the program directory. Exiting.",
			zenity.Title(mBoxTitle),
		)
		os.Exit(1)
	}

	systray.SetIcon(systrayIcon)
	if runtime.GOOS == "windows" {
		systray.SetTitle("WirePod")
	}
	systray.SetTooltip("WirePod is starting...")
	mQuit := systray.AddMenuItem("Quit", "Quit WirePod")
	mBrowse := systray.AddMenuItem("Web Interface", "Open web UI")
	mConfig := systray.AddMenuItem("Config Folder", "Open config folder in case you need to. The web UI should have everything you need.")
	mStartup := systray.AddMenuItem("Run On Startup", "")
	mAbout := systray.AddMenuItem("About", "About WirePod")

	conf, _ := cross.ReadConfig()
	if conf.RunAtStartup {
		mStartup.Check()
	} else {
		mStartup.Uncheck()
	}

	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				zenity.Info(
					"WirePod will now exit.",
					zenity.Icon(mBoxIcon()),
					zenity.Title(mBoxTitle),
				)
				ExitProgram(0)
			case <-mBrowse.ClickedCh:
				go openBrowser("http://" + vars.GetOutboundIP().String() + ":" + vars.WebPort)
			case <-mConfig.ClickedCh:
				conf, _ := os.UserConfigDir()
				go openFileExplorer(filepath.Join(conf, vars.PodName))
			case <-mAbout.ClickedCh:
				zenity.Info("WirePod is an Escape Pod alternative which is able to get any Anki/DDL Vector robot setup and working with voice commands.\n\nVersion: "+conf.Version,
					zenity.Icon(mBoxIcon()),
					zenity.Title("WirePod"))
			case <-mStartup.ClickedCh:
				if mStartup.Checked() {
					mStartup.Uncheck()
					cross.RunPodAtStartup(false)
				} else {
					mStartup.Check()
					cross.RunPodAtStartup(true)
				}
			}
		}
	}()

	StartFromProgramInit(stt.Init, stt.STT, stt.Name)
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		go zenity.Warning(
			"Error opening browser: "+err.Error(),
			zenity.WarningIcon,
			zenity.Title(mBoxTitle),
		)
		logger.Println(err)
	}
}

func openFileExplorer(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
