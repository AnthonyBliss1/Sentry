package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	deploy "github.com/anthonybliss1/Sentry/Hub/Deploy"
	network "github.com/anthonybliss1/Sentry/Hub/Network"
	services "github.com/anthonybliss1/Sentry/Hub/Services"
	utils "github.com/anthonybliss1/Sentry/Hub/Utils"
)

var hub network.Hub

func init() {
	var err error

	LanIP, err := utils.LANIPv4()
	if err != nil {
		log.Fatalf("Could not get LAN IP: %q", err)
	}

	hostName, err := os.Hostname()
	if err != nil {
		log.Fatalf("Could not get hostname: %q", err)
	}

	hub.LanIP = LanIP
	hub.Hostname = hostName
}

func main() {
	d := flag.Bool("deploy", false, "creates systemD or launchD file depending on OS")

	flag.Parse()

	if *d {
		if err := deploy.DeployHub(); err != nil {
			utils.Red.Printf("failed to deploy hub: %q\n", err)
			return
		}

		utils.Blue.Println("\nSentry Hub successfully deployed!")
		return
	}

	if hub.LanIP.String() == "" || hub.Hostname == "" {
		log.Fatal("LanIP and Hostname must be set")
	}

	// need to set ip env var for mediamtx docker container
	if err := os.Setenv("LAN_IP", hub.LanIP.String()); err != nil {
		log.Fatal(err)
	}

	// create service containers
	var CC services.ContainerController

	// create mediamtx & obj detection services
	mtxService := services.Service{Name: "sentry-hub-mediamtx", ComposeYML: "mtx-compose.yml", Files: []string{"mtx-compose.yml", "mediamtx.yml"}}
	objDetectService := services.Service{Name: "sentry-hub-object-detection", ComposeYML: "obj-detect-compose.yml", Files: []string{"obj-detect-compose.yml", "Dockerfile.detect", "detect-object.py"}}

	CC.Containers = append(CC.Containers, &services.Container{Service: &mtxService}, &services.Container{Service: &objDetectService})

	// create the composers for all services
	if err := CC.CreateAllComposers(); err != nil {
		log.Fatal(err)
	}

	// start the service composers
	utils.Blue.Println("> Starting service containers...")
	if err := CC.StartAllServices(); err != nil {
		log.Fatal(err)
	}

	// context that is cancelled on ctrl+c, sigTerm
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hub.Concierge = network.Concierge{
		RTSPPublishBase: fmt.Sprintf("rtsp://%s:8554", hub.LanIP),
		WebRTCBase:      fmt.Sprintf("http://%s:8889", hub.LanIP),
		HLSBase:         fmt.Sprintf("http://%s:8888", hub.LanIP),
	}

	// start http server for sharing room info
	utils.Blue.Println("> Starting Concierge Service...")
	hub.StartConciergeService()

	// start ws server for delivering commands to camera
	utils.Blue.Println("> Starting Commander Service...")
	hub.StartCommanderService()

	// start mDNS server for service discovery
	utils.Blue.Println("> Starting MDNS...")
	hub.StartMDNS()

	// wait for ctrl+c
	<-ctx.Done() // blocking

	utils.Blue.Println("\n> Taking down MDNS...")
	if hub.HTTPMDNS != nil {
		hub.HTTPMDNS.Shutdown()
	}

	if hub.WSMDNS != nil {
		hub.WSMDNS.Shutdown()
	}

	utils.Blue.Println("> Taking down Concierge Service...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// SSE
	if hub.Detections != nil {
		hub.Detections.Shutdown()
	}

	if err := hub.HTTPSrv.Shutdown(shutdownCtx); err != nil {
		utils.Red.Println(err)
	}

	utils.Blue.Println("> Taking down Commander Service...")
	if hub.Commander != nil {
		hub.Shutdown() // calls Commander.Shutdown()
	}

	utils.Blue.Println("> Taking down service containers...")
	if err := CC.StopAllServices(); err != nil {
		utils.Red.Println(err)
	}

	utils.Green.Println("[ Program exited gracefully ]")
}
