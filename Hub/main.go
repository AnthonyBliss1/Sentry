package main

import (
	"fmt"
	"log"
	"os"

	network "github.com/anthonybliss1/Sentry/Hub/Network"
	utils "github.com/anthonybliss1/Sentry/Hub/Utils"
)

var (
	hub    network.Hub
	hlsDir string
)

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

	hlsDir, err = utils.ValidateSaveDir()
	if err != nil {
		log.Fatal(err)
	}

	hub.LanIP = LanIP
	hub.Hostname = hostName
}

func main() {
	// safeguard against potential empty fields
	if hub.LanIP.String() == "" || hub.Hostname == "" {
		log.Fatal("LanIP and Hostname must be set")
	}

	utils.Blue.Println("> Running livekit-server --dev...")
	if err := hub.StartLKServer(); err != nil {
		log.Fatal(err)
	}

	// setup LKRoom
	utils.Blue.Println("> Creating LK Room...")
	hub.SetRoom("Sentry-Hub", fmt.Sprintf("ws://%s:%d", hub.LanIP, 7880))

	// start http server for sharing room info
	utils.Blue.Println("> Starting Room Service API...")
	hub.StartRoomService()

	// start mDNS server for service discovery
	utils.Blue.Println("> Starting MDNS...")
	hub.StartMDNS()

	// persist forever
	select {}
}
