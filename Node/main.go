package main

import (
	"log"
	"time"

	client "github.com/anthonybliss1/Sentry/Node/Client"
	video "github.com/anthonybliss1/Sentry/Node/Video"
	"github.com/fatih/color"
)

var (
	green = color.New(color.FgGreen)
	red   = color.New(color.FgRed)
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
	hlsDir, err := video.ValidateOutputDir()
	if err != nil {
		log.Fatal(err)
	}

	foundServers := false
	node := client.NodeClient{}

	// Look for hub servers until both are found
	for !foundServers {
		if err := node.MDNSLookup(); err != nil {
			log.Fatal(err)
		}

		node.Mu.Lock()
		foundServers = node.Hub.WS != (client.Websocket{}) && node.Hub.FS != (client.FileServer{})
		node.Mu.Unlock()
	}

	// initialize file agent to watch hlsDir
	green.Println("> Deploying Watchdog...")
	go client.DeployWatchdog(&node, hlsDir)

	// start recording and creating segments
	green.Println("> Starting Video Stream...")
	stream, err := video.StartStream(hlsDir)
	if err != nil {
		red.Print(err)
	}

	// wait a bit
	time.Sleep(120 * time.Second)

	// test stop
	if err := stream.Stop(); err != nil {
		red.Print(err)
	}
}
