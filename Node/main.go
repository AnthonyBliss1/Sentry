package main

import (
	"log"
	"net/http"
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

	// validate dependencies (FFMpeg and camera libs)
	//
	// if err := CheckDeps(); err != nil {
	// log.Fatal(err)
	// }

	// validate outputDir for video segments
	hlsDir, err := utils.ValidateOutputDir()
	if err != nil {
		log.Fatal(err) // should kill program if no valid outputDir
	}

	// find Hub services for (WS and FS)
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	foundServers := false

	for !foundServers {
		if err := node.MDNSLookup(); err != nil {
			log.Fatal(err)
		}

		// check if both services have been found
		node.Mu.Lock()
		foundServers = (node.WS != (network.Websocket{}) && node.FS != (network.FileServer{}))
		node.Mu.Unlock()

		if !foundServers {
			time.Sleep(500 * time.Millisecond)
		}
	}

	// initialize client for FS
	node.FS.Client = &http.Client{}

	// initialize file agent to watch hlsDir
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	utils.Blue.Println("> Deploying Watchdog...")
	go func() {
		if err := network.DeployWatchdog(&node, hlsDir); err != nil {
			log.Fatal(err)
		}
	}()

	// start recording and creating segments
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	utils.Blue.Println("> Starting Video Stream...")
	if err := stream.Start(hlsDir); err != nil {
		utils.Red.Print(err)
	}

	// test end
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	time.Sleep(120 * time.Second)

	if err := stream.Stop(); err != nil {
		utils.Red.Print(err)
	}
}
