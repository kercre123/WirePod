package main

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/kercre123/wire-pod/chipper/pkg/initwirepod"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/mdnshandler"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	wirepod_vosk "github.com/kercre123/wire-pod/chipper/pkg/wirepod/stt/vosk"
	"github.com/wlynxg/anet"
)

var DataPath string

func IsConnedToWifi() bool {
	ifaces, _ := anet.Interfaces()
	for _, iface := range ifaces {
		if iface.Name == "wlan0" {
			return true
		}
	}
	return false
}

func main() {
	myApp := app.New()
	myApp.Settings().SetTheme(theme.DarkTheme())
	DataPath = filepath.Dir(myApp.Storage().RootURI().Path())
	logger.Println("DATAPATH: " + DataPath)
	version := myApp.Metadata().Version
	if NeedUnzip(version) {
		fmt.Println("Unzipping static content")
		DeleteStaticContent()
		DoUnzip()
	}
	vars.AndroidPath = DataPath
	vars.Packaged = true
	PodWindow(myApp)
	myApp.Run()
}

func PodWindow(myApp fyne.App) {

	window := myApp.NewWindow("pod")
	window.SetMaster()

	var stuffContainer fyne.CanvasObject

	firstCard := widget.NewCard("WirePod", "", container.NewWithoutLayout())

	exitButton := widget.NewButton("Exit", func() {
		os.Exit(0)
	})

	contextCheck := widget.NewCheck("with specific grammer?", func(checked bool) {
		if checked {
			wirepod_vosk.GrammerEnable = true
		} else {
			wirepod_vosk.GrammerEnable = false
		}
	})

	var linkLabel *widget.RichText
	linkLabel = widget.NewRichTextWithText("Configuration/setup page:")
	linkLabel.Hide()

	var hyprLink *widget.Hyperlink
	hyprLink = widget.NewHyperlink("test", &url.URL{
		Scheme: "http",
		Host:   "",
	})
	hyprLink.Hide()

	secondCard := widget.NewCard("WirePod Control", "", container.NewWithoutLayout())
	var startButton *widget.Button
	startButton = widget.NewButton("Start", func() {
		if !IsConnedToWifi() {
			dialog.ShowCustom("This device must be connected to Wi-Fi first", "OK", container.NewWithoutLayout(), window)
			return
		}
		secondCard.SetSubTitle("Running!")
		go func() {
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			go mdnshandler.PostmDNS()
			go func() {
				PingJdocsInit()
				PingJdocsStart()
			}()
			hyprLink.SetText("http://" + vars.GetOutboundIP().String() + ":8080")
			hyprLink.SetURL(&url.URL{
				Scheme: "http",
				Host:   vars.GetOutboundIP().String() + ":8080",
			})
			hyprLink.Show()
			linkLabel.Show()
			startButton.Disable()
			contextCheck.Disable()
			initwirepod.StartFromProgramInit(wirepod_vosk.Init, wirepod_vosk.STT, wirepod_vosk.Name)
			startButton.Enable()
			contextCheck.Enable()
			hyprLink.Hide()
			linkLabel.Hide()
			secondCard.SetSubTitle("wirepod failed :(")
		}()
	})

	stuffContainer = container.NewVScroll(container.NewVBox(
		firstCard,
		exitButton,
		widget.NewSeparator(),
		secondCard,
		linkLabel,
		hyprLink,
		contextCheck,
		startButton,
	))

	window.SetContent(stuffContainer)

	window.Show()
}

func DeleteStaticContent() {
	os.RemoveAll(filepath.Join(DataPath, "/static"))
}

func NeedUnzip(version string) bool {
	versionFilePath := filepath.Join(DataPath, "/static/version")
	versionFileBytes, err := os.ReadFile(versionFilePath)
	if err != nil {
		return true
	}
	fmt.Println(version, string(versionFileBytes))
	if strings.TrimSpace(version) == strings.TrimSpace(string(versionFileBytes)) {
		return false
	}
	return true
}

func DoUnzip() {
	UnzipBytes(resourceStaticZip.Content(), filepath.Join(DataPath, "/static/"))
}

func UnzipBytes(zipBytes []byte, destDir string) error {
	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		logger.Println(file.Name)
		path := filepath.Join(destDir, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, os.ModePerm)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
