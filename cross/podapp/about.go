package podapp

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var AboutWindow fyne.Window
var WindowDefined bool

var FakeWindow fyne.Window

func IsPodRunningAtStartup() bool {
	conf, _ := cross.ReadConfig()
	return conf.RunAtStartup
}

func GetPodVersion() string {
	conf, err := cross.ReadConfig()
	if err != nil {
		return "error getting version =("
	}
	return conf.Version
}

func DefineAboutWindow(myApp fyne.App) {
	AboutWindow = myApp.NewWindow("wire-pod")
	AboutWindow.Resize(fyne.Size{Width: 400, Height: 100})
	AboutWindow.CenterOnScreen()
	icon, err := os.ReadFile(mBoxIcon)
	var iconRes *fyne.StaticResource
	if err == nil {
		iconRes = fyne.NewStaticResource("podIcon", icon)
		AboutWindow.SetIcon(iconRes)
	} else {
		fmt.Println("error loading icon: " + fmt.Sprint(err))
	}
	card := widget.NewCard("wire-pod", "wire-pod is an Escape Pod alternative which is able to get any Anki/DDL Vector robot setup and working with voice commands.",
		container.NewWithoutLayout())

	version := widget.NewRichTextWithText("Version: " + GetPodVersion())

	runStartup := widget.NewCheck("Run wire-pod when user logs in?", func(checked bool) {
		cross.RunPodAtStartup(checked)
	})

	runStartup.SetChecked(IsPodRunningAtStartup())

	exitButton := widget.NewButton("Close", func() {
		AboutWindow.Hide()
	})

	AboutWindow.SetContent(container.NewVBox(
		card,
		version,
		runStartup,
		widget.NewSeparator(),
		exitButton,
	))

}

func ShowAbout(myApp fyne.App) {
	if !WindowDefined {
		FakeWindow = myApp.NewWindow("a")
		FakeWindow.Hide()
		DefineAboutWindow(myApp)
	}

	FakeWindow.SetMaster()
	AboutWindow.Show()
}
