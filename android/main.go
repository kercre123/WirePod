package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/kercre123/wire-pod/chipper/pkg/initwirepod"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	botsetup "github.com/kercre123/wire-pod/chipper/pkg/wirepod/setup"
	wirepod_vosk "github.com/kercre123/wire-pod/chipper/pkg/wirepod/stt/vosk"
	"github.com/kercre123/zeroconf"
	"github.com/wlynxg/anet"
)

var PostingMDNS bool

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
	go PostmDNSWhenNewVector()
	go func() {
		PingJdocsInit()
		PingJdocsStart()
	}()
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

	secondCard := widget.NewCard("WirePod Control", "", container.NewWithoutLayout())
	var startButton *widget.Button
	startButton = widget.NewButton("Start", func() {
		if !IsConnedToWifi() {
			dialog.ShowCustom("this device must be connected to Wi-Fi first", "OK", container.NewWithoutLayout(), window)
			return
		}
		secondCard.SetSubTitle("running! http://" + botsetup.GetOutboundIP().String() + ":8080")
		go func() {
			startButton.Disable()
			contextCheck.Disable()
			initwirepod.StartFromProgramInit(wirepod_vosk.Init, wirepod_vosk.STT, wirepod_vosk.Name)
			startButton.Enable()
			contextCheck.Enable()
			secondCard.SetSubTitle("wirepod failed :(")
		}()
	})

	stuffContainer = container.NewVScroll(container.NewVBox(
		firstCard,
		exitButton,
		widget.NewSeparator(),
		secondCard,
		contextCheck,
		startButton,
	))

	window.SetContent(stuffContainer)

	window.Show()
}

func PostmDNS() error {
	if PostingMDNS {
		return nil
	}
	PostingMDNS = true
	logger.Println("Registering escapepod.local on network (every minute)")
	mdnsport := 8084
	for {
		ipAddr := botsetup.GetOutboundIP().String()
		server, _ := zeroconf.RegisterProxy("escapepod", "_app-proto._tcp", "local.", mdnsport, "escapepod", []string{ipAddr}, []string{"txtv=0", "lo=1", "la=2"}, nil)
		time.Sleep(time.Second * 10)
		server.Shutdown()
		server = nil
		time.Sleep(time.Second * 2)
	}
}

// what if i constantly have an mDNS browser, and post on the network right when it sees a new vector on the net?

func PostmDNSWhenNewVector() {
	for {
		resolver, _ := zeroconf.NewResolver(nil)
		entries := make(chan *zeroconf.ServiceEntry)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
		err := resolver.Browse(ctx, "_ankivector._tcp", "local.", entries)
		if err != nil {
			fmt.Println(err)
			return
		}
		for entry := range entries {
			if strings.Contains(entry.Service, "ankivector") {
				logger.Println("New vector discovered on the network! posting mDNS...")
				time.Sleep(time.Second)
				go PostmDNS()
				defer cancel()
				return
			}
		}
		cancel()
		logger.Println("done")
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
	fmt.Println(currentVersion, string(versionFileBytes))
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
