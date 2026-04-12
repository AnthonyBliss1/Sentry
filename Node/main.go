package main

import (
	"log"
	"time"

	network "github.com/anthonybliss1/Sentry/Node/Network"
	utils "github.com/anthonybliss1/Sentry/Node/Utils"
	video "github.com/anthonybliss1/Sentry/Node/Video"
)

var (
	node   network.NodeClient
	stream video.Stream
)

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
	foundServers := false

	for !foundServers {
		if err := node.MDNSLookup(); err != nil {
			log.Fatal(err)
		}

		node.Mu.Lock()
		foundServers = node.TCP != (network.TCPServer{})
		node.Mu.Unlock()

		if !foundServers {
			time.Sleep(500 * time.Millisecond)
		}
	}

	// start recording and creating segments
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	utils.Blue.Println("> Starting Video Stream...")
	if err := stream.Start(node.TCP.URL); err != nil {
		utils.Red.Print(err)
	}

	// test end
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	time.Sleep(120 * time.Second)

	if err := stream.Stop(); err != nil {
		utils.Red.Print(err)
	}
}
