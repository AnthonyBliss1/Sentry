package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	deploy "github.com/anthonybliss1/Sentry/Hub/Deploy"
	network "github.com/anthonybliss1/Sentry/Hub/Network"
	utils "github.com/anthonybliss1/Sentry/Hub/Utils"
	"github.com/docker/compose/v5/pkg/api"
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

	// create composer
	composer, err := utils.CreateComposer()
	if err != nil {
		log.Fatal(err)
	}

	// start the composer
	utils.Blue.Println("> Starting MediaMTX container...")
	if err := composer.Service.Up(composer.Ctx, composer.Project, api.UpOptions{}); err != nil {
		log.Fatal(err)
	}

	// context that is cancelled on ctrl+c, sigTerm, or sigKill
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer stop()

	hub.Concierge = network.Concierge{
		RTSPPublishBase: fmt.Sprintf("rtsp://%s:8554", hub.LanIP),
		WebRTCBase:      fmt.Sprintf("http://%s:8889", hub.LanIP),
		HLSBase:         fmt.Sprintf("http://%s:8888", hub.LanIP),
	}

	// start http server for sharing room info
	utils.Blue.Println("> Starting Room Service API...")
	hub.StartRoomService()

	// start mDNS server for service discovery
	utils.Blue.Println("> Starting MDNS...")
	hub.StartMDNS()

	// wait for ctrl+c
	<-ctx.Done() // blocking
	utils.Blue.Println("\n> Taking down MediaMTX container...")
	if err := composer.Service.Down(composer.Ctx, composer.Project.Name, api.DownOptions{Images: "all"}); err != nil {
		utils.Red.Println(err)
	}

	utils.Green.Println("[ Program exited gracefully ]")
}
