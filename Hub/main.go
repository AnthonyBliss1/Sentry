package main

import (
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

	// start TCP server
	utils.Blue.Println("> Starting TCP...")
	hub.StartTCP(hlsDir)

	// start FS server
	utils.Blue.Println("> Starting FS...")
	hub.StartFS(hlsDir)

	// start mDNS server for service discovery
	utils.Blue.Println("> Starting MDNS...")
	hub.StartMDNS()

	select {}
}
