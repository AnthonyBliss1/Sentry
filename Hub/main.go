package main

import (
	"log"
	"os"

	network "github.com/anthonybliss1/Sentry/Hub/Network"
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
	// safeguard against potential empty fields
	if hub.LanIP.String() == "" || hub.Hostname == "" {
		log.Fatal("LanIP and Hostname must be set")
	}

	// start WS server
	utils.Blue.Println("> Starting FS...")
	hub.StartFS()

	// start FS
	utils.Blue.Println("> Starting WS...")
	hub.StartWS()

	// start mDNS server for service discovery
	utils.Blue.Println("> Starting MDNS...")
	hub.StartMDNS()

	select {}
}
