package main

import (
	"log"
	"os"
	"time"

	network "github.com/anthonybliss1/Sentry/Node/Network"
	utils "github.com/anthonybliss1/Sentry/Node/Utils"
)

var node network.NodeClient

// stream video.Stream

func main() {
	// TODO: implement these checks
	//
	// should run checks for camera setup
	//
	// if err := InitCamera(); err != nil {
	// log.Fatal(err)
	// }

	// validate dependencies (FFMpeg? and camera libs)
	//
	// if err := CheckDeps(); err != nil {
	// log.Fatal(err)
	// }

	// find Hub services
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	utils.Blue.Println("> Looking for Room Service...")
	node.RoomServiceLookup()

	utils.Blue.Println("> Fetching Concierge...")
	if err := node.FetchConcierge(); err != nil {
		log.Fatal(err)
	}

	deviceID, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}

	// start recording and creating segments
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	utils.Blue.Println("> Publishing Video Stream...")
	stream, err := node.PublishStream(deviceID)
	if err != nil {
		log.Fatal(err)
	}

	// test end
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	time.Sleep(120 * time.Second)

	if err := stream.Stop(); err != nil {
		utils.Red.Print(err)
	}
}
