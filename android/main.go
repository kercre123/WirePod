package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/kercre123/chipper/pkg/initwirepod"
	"github.com/kercre123/chipper/pkg/logger"
	"github.com/kercre123/chipper/pkg/vars"
	botsetup "github.com/kercre123/chipper/pkg/wirepod/setup"
	wirepod_vosk "github.com/kercre123/chipper/pkg/wirepod/stt/vosk"
	"github.com/kercre123/zeroconf"
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
	DataPath = filepath.Dir(myApp.Storage().RootURI().Path())
	logger.Println("DATAPATH: " + DataPath)
	if NeedUnzip() {
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

	window := myApp.NewWindow("test")
	window.SetMaster()

	firstCard := widget.NewCard("WirePod", "", container.NewWithoutLayout())

	exitButton := widget.NewButton("Exit", func() {
		os.Exit(0)
	})

	logText := widget.NewLabel(logger.LogTrayList)
	scrollContainer := container.NewVScroll(logText)
	scrollContainer.Resize(fyne.NewSize(1000, 1000))

	secondCard := widget.NewCard("status", "", container.NewWithoutLayout())
	var startButton *widget.Button
	startButton = widget.NewButton("Start", func() {
		if !IsConnedToWifi() {
			secondCard.SetSubTitle("this device must be connected to wifi first")
			return
		}
		secondCard.SetSubTitle("running! http://" + botsetup.GetOutboundIP().String() + ":8080")
		go func() {
			startButton.Disable()
			initwirepod.StartFromProgramInit(wirepod_vosk.Init, wirepod_vosk.STT, wirepod_vosk.Name)
			startButton.Enable()
			secondCard.SetSubTitle("wirepod failed :(")
		}()
	})

	fourthCard := widget.NewCard("broadcast mDNS", "", container.NewWithoutLayout())

	var mdnsButton *widget.Button
	mdnsButton = widget.NewButton("start broadcasting", func() {
		fourthCard.SetSubTitle("broadcasting")
		go func() {
			mdnsButton.Disable()
			err := PostmDNS()
			mdnsButton.Enable()
			if err != nil {
				fourthCard.SetSubTitle(err.Error())
			}
		}()
	})

	fifthCard := widget.NewCard("ping Jdocs", "", container.NewWithoutLayout())

	var jdocsButton *widget.Button
	jdocsButton = widget.NewButton("start pinging jdocs", func() {
		PingJdocsInit()
		go func() {
			fifthCard.SetSubTitle("starting Jdocs pinger...")
			PingJdocsStart()
		}()
		go func() {
			timeChan := GetTimeChannel()
			for time := range timeChan {
				if time == 0 {
					fifthCard.SetSubTitle("pinging...")
				} else {
					fifthCard.SetSubTitle(fmt.Sprint(time) + " secs until next ping")
				}
			}
		}()
		jdocsButton.Disable()
	})

	var jdocsNowButton *widget.Button
	jdocsNowButton = widget.NewButton("ping Jdocs now", func() {
		go func() {
			fifthCard.SetSubTitle("pinging...")
			jdocsNowButton.Disable()
			PingNow()
			fifthCard.SetSubTitle("pinging done")
			jdocsNowButton.Enable()
		}()
	})

	window.SetContent(container.NewVScroll(container.NewVBox(
		firstCard,
		exitButton,
		widget.NewSeparator(),
		secondCard,
		startButton,
		widget.NewSeparator(),
		fourthCard,
		mdnsButton,
		widget.NewSeparator(),
		fifthCard,
		jdocsButton,
		jdocsNowButton,
	)))

	window.Show()
}

func PostmDNS() error {
	logger.Println("Registering escapepod.local on network (every minute)")
	mdnsport := 8084

	ipAddr := botsetup.GetOutboundIP().String()
	// create a server a few times to ensure broadcast
	for i := 0; i < 3; i++ {
		server, err := zeroconf.RegisterProxy("escapepod", "_app-proto._tcp", "local.", mdnsport, "escapepod", []string{ipAddr}, []string{"txtv=0", "lo=1", "la=2"}, nil)
		if err != nil {
			return err
		}
		time.Sleep(time.Second / 3)
		server.Shutdown()
	}

	time.Sleep(time.Second)

	for {
		ipAddr := botsetup.GetOutboundIP().String()
		server, err := zeroconf.RegisterProxy("escapepod", "_app-proto._tcp", "local.", mdnsport, "escapepod", []string{ipAddr}, []string{"txtv=0", "lo=1", "la=2"}, nil)
		if err != nil {
			return err
		}
		time.Sleep(time.Second * 60)
		server.Shutdown()
		server = nil
		time.Sleep(time.Second * 2)
	}
}

func DeleteStaticContent() {
	os.RemoveAll(filepath.Join(DataPath, "/static"))
}

func NeedUnzip() bool {
	currentVersion := fyne.CurrentApp().Metadata().Version
	versionFilePath := filepath.Join(DataPath, "/static/version")
	versionFileBytes, err := os.ReadFile(versionFilePath)
	if err != nil {
		return true
	}
	if strings.TrimSpace(currentVersion) == strings.TrimSpace(string(versionFileBytes)) {
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
