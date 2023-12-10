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
	// if NeedsRestart && hostname != escapepod; then error
	NeedsRestart   bool `json:"needsrestart"`
	LastRunningPID int  `json:"lastrunningpid"`
	FirstStartup   bool `json:"firststartup"`
	NoPodWarn      bool `json:"nopodwarn"`
}

type OSFuncs interface {
	Init() error
	ReadConfig() (WPConfig, error)
	RunPodAtStartup(bool) error
	WriteConfig(WPConfig) error
	IsPodAlreadyRunning() bool
	IsPIDProcessRunning(int) (bool, error)
	KillExistingPod() error
	ResourcesPath() string
	Hostname() string
	OnExit()
}
