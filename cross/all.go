package cross

/*
Function definitions:

func Init() err
func ReadConfig() (WPConfig, err)
func RunAtStartup(bool) err
func WriteConfig(WPConfig) err
func IsPodAlreadyRunning() bool
func KillExistingPod() err
func OnExit()

Notes:

Use zenity for message boxes.
Use fyne for about window if possible ("Check for updates" planned for the future)
*/

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
