package podapp

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	chipperpb "github.com/digital-dream-labs/api/go/chipperpb"
	"github.com/digital-dream-labs/api/go/jdocspb"
	"github.com/digital-dream-labs/api/go/tokenpb"
	"github.com/digital-dream-labs/hugh/log"
	"github.com/getlantern/systray"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/mdnshandler"
	chipperserver "github.com/kercre123/wire-pod/chipper/pkg/servers/chipper"
	jdocsserver "github.com/kercre123/wire-pod/chipper/pkg/servers/jdocs"
	tokenserver "github.com/kercre123/wire-pod/chipper/pkg/servers/token"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	wpweb "github.com/kercre123/wire-pod/chipper/pkg/wirepod/config-ws"
	wp "github.com/kercre123/wire-pod/chipper/pkg/wirepod/preqs"
	sdkWeb "github.com/kercre123/wire-pod/chipper/pkg/wirepod/sdkapp"
	"github.com/ncruces/zenity"
	"github.com/soheilhy/cmux"

	//	grpclog "github.com/digital-dream-labs/hugh/grpc/interceptors/logger"

	grpcserver "github.com/digital-dream-labs/hugh/grpc/server"
)

var serverOne cmux.CMux
var serverTwo cmux.CMux
var listenerOne net.Listener
var listenerTwo net.Listener
var voiceProcessor *wp.Server

var NotSetUp string = "Wire-pod is not setup. Use the webserver at port " + vars.WebPort + " to set up wire-pod."

func NeedsSetupMsg() {
	go func() {
		err := zenity.Info(
			getNeedsSetupMsg(),
			zenity.Icon(mBoxIcon()),
			zenity.Title(mBoxTitle),
			zenity.ExtraButton("Open browser"),
			zenity.OKLabel("OK"),
		)
		if err != nil {
			if err == zenity.ErrExtraButton {
				openBrowser("http://" + vars.GetOutboundIP().String() + ":" + vars.WebPort)
			}
		}
	}()
}

func ErrMsg(err error) {
	zenity.Error("wire-pod has run into an issue. The program will now exit. Error details: "+err.Error(),
		zenity.ErrorIcon,
		zenity.Title(mBoxTitle))
	ExitProgram(1)
}

// grpcServer *grpc.Servervar
var chipperServing bool = false

func serveOk(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok")
}

func httpServe(l net.Listener) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok:80", serveOk)
	mux.HandleFunc("/ok", serveOk)
	s := &http.Server{
		Handler: mux,
	}
	return s.Serve(l)
}

func grpcServe(l net.Listener, p *wp.Server) error {
	srv, err := grpcserver.New(
		grpcserver.WithViper(),
		grpcserver.WithReflectionService(),
		grpcserver.WithInsecureSkipVerify(),
	)
	if err != nil {
		log.Fatal(err)
	}

	s, _ := chipperserver.New(
		chipperserver.WithIntentProcessor(p),
		chipperserver.WithKnowledgeGraphProcessor(p),
		chipperserver.WithIntentGraphProcessor(p),
	)

	tokenServer := tokenserver.NewTokenServer()
	jdocsServer := jdocsserver.NewJdocsServer()
	//jdocsserver.IniToJson()

	chipperpb.RegisterChipperGrpcServer(srv.Transport(), s)
	jdocspb.RegisterJdocsServer(srv.Transport(), jdocsServer)
	tokenpb.RegisterTokenServer(srv.Transport(), tokenServer)

	return srv.Transport().Serve(l)
}

func BeginWirepodSpecific(sttInitFunc func() error, sttHandlerFunc interface{}, voiceProcessorName string) error {
	logger.Init()

	// begin wirepod stuff
	vars.Init()
	var err error
	voiceProcessor, err = wp.New(sttInitFunc, sttHandlerFunc, voiceProcessorName)
	wpweb.SttInitFunc = sttInitFunc
	go sdkWeb.BeginServer()
	http.HandleFunc("/api-chipper/", ChipperHTTPApi)
	if err != nil {
		return err
	}
	return nil
}

func StartFromProgramInit(sttInitFunc func() error, sttHandlerFunc interface{}, voiceProcessorName string) {
	err := BeginWirepodSpecific(sttInitFunc, sttHandlerFunc, voiceProcessorName)
	if err != nil {
		logger.Println("Wire-pod is not setup. Use the webserver at port " + vars.WebPort + " to set up wire-pod.")
		vars.APIConfig.PastInitialSetup = false
		vars.WriteConfigToDisk()
		NeedsSetupMsg()
		systray.SetTooltip("wire-pod must be set up at http://" + vars.GetOutboundIP().String() + ":" + vars.WebPort)
	} else if !vars.APIConfig.PastInitialSetup {
		logger.Println("Wire-pod is not setup. Use the webserver at port " + vars.WebPort + " to set up wire-pod.")
		NeedsSetupMsg()
		systray.SetTooltip("wire-pod must be set up at http://" + vars.GetOutboundIP().String() + ":" + vars.WebPort)
	} else if vars.APIConfig.STT.Service == "vosk" && vars.APIConfig.STT.Language == "" {
		logger.Println("\033[33m\033[1mLanguage value is blank, but STT service is Vosk. Reinitiating setup process.\033[0m")
		logger.Println("Wire-pod is not setup. Use the webserver at port " + vars.WebPort + " to set up wire-pod.")
		NeedsSetupMsg()
		systray.SetTooltip("wire-pod must be set up at http://" + vars.GetOutboundIP().String() + ":" + vars.WebPort)
		vars.APIConfig.PastInitialSetup = false
	} else {
		go StartChipper(true)
	}
	// main thread is configuration ws
	wpweb.StartWebServer()
}

func IfFileExist(name string) bool {
	_, err := os.Stat(name)
	if err != nil {
		return false
	}
	return true
}

func RestartServer() {
	if chipperServing {
		serverOne.Close()
		serverTwo.Close()
		listenerOne.Close()
		listenerTwo.Close()
	}
	go StartChipper(false)
}

func StartChipper(fromInit bool) {
	if vars.APIConfig.Server.EPConfig {
		go mdnshandler.PostmDNS()
	}
	// load certs
	var certPub []byte
	var certPriv []byte
	if vars.APIConfig.Server.EPConfig {
		certPub, _ = os.ReadFile("./epod/ep.crt")
		certPriv, _ = os.ReadFile("./epod/ep.key")
		vars.ChipperKey = certPriv
		vars.ChipperCert = certPub
	} else {
		if !vars.ChipperKeysLoaded {
			var err error
			certPub, _ = os.ReadFile(vars.CertPath)
			certPriv, err = os.ReadFile(vars.KeyPath)
			if err != nil {
				logger.Println("Unable to read certificates. wire-pod is not setup.")
				logger.Println(err)
				ErrMsg(err)
				ExitProgram(1)
				return
			}
			vars.ChipperKey = certPriv
			vars.ChipperCert = certPub
		}
	}

	logger.Println("Initiating TLS listener, cmux, gRPC handler, and REST handler")
	cert, err := tls.X509KeyPair(vars.ChipperCert, vars.ChipperKey)
	if err != nil {
		ErrMsg(err)
		logger.Println(err)
		ExitProgram(1)
	}
	logger.Println("Starting chipper server at port " + vars.APIConfig.Server.Port)
	listenerOne, err = tls.Listen("tcp", ":"+vars.APIConfig.Server.Port, &tls.Config{
		Certificates: []tls.Certificate{cert},
		CipherSuites: nil,
	})
	if err != nil {
		ErrMsg(err)
		fmt.Println(err)
		ExitProgram(1)
	}
	serverOne = cmux.New(listenerOne)
	grpcListenerOne := serverOne.Match(cmux.HTTP2())
	httpListenerOne := serverOne.Match(cmux.HTTP1Fast())
	go grpcServe(grpcListenerOne, voiceProcessor)
	go httpServe(httpListenerOne)

	if vars.APIConfig.Server.EPConfig && os.Getenv("NO8084") != "true" {
		logger.Println("Starting chipper server at port 8084 for 2.0.1 compatibility")
		listenerTwo, err = tls.Listen("tcp", ":8084", &tls.Config{
			Certificates: []tls.Certificate{cert},
			CipherSuites: nil,
		})
		if err != nil {
			ErrMsg(err)
			fmt.Println(err)
			ExitProgram(1)
		}
		serverTwo = cmux.New(listenerTwo)
		grpcListenerTwo := serverTwo.Match(cmux.HTTP2())
		httpListenerTwo := serverTwo.Match(cmux.HTTP1Fast())
		go grpcServe(grpcListenerTwo, voiceProcessor)
		go httpServe(httpListenerTwo)
	}

	systray.SetTooltip("wire-pod is running.\n" + "http://" + vars.GetOutboundIP().String() + ":" + vars.WebPort)
	var discrete bool
	if len(os.Args) > 1 {
		if strings.Contains(os.Args[1], "-d") {
			discrete = true
		}
	}
	if fromInit && !discrete {
		go zenity.Info(
			mBoxSuccess,
			zenity.Icon(mBoxIcon()),
			zenity.Title(mBoxTitle),
		)
	}
	fmt.Println("\033[33m\033[1mwire-pod started successfully!\033[0m")

	chipperServing = true
	if vars.APIConfig.Server.EPConfig && os.Getenv("NO8084") != "true" {
		go serverOne.Serve()
		serverTwo.Serve()
		logger.Println("Stopping chipper server")
		chipperServing = false
	} else {
		serverOne.Serve()
		logger.Println("Stopping chipper server")
		chipperServing = false
	}
}
