package main

import (
	"log"
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

	utils.Blue.Println("> Joining Room...")
	if err := node.JoinRoom(); err != nil {
		log.Fatal(err)
	}

	time.Sleep(10 * time.Second)

	utils.Blue.Println("> Leaving Room...")
	if ok := node.LeaveRoom(); !ok {
		utils.Red.Println("[ Failed to leave room! ]")
	}

	// start recording and creating segments
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	//	utils.Blue.Println("> Starting Video Stream...")
	//	if err := stream.Start(node.TCP.URL); err != nil {
	//		utils.Red.Print(err)
	//	}
	//
	//	// test end
	//	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	//	time.Sleep(120 * time.Second)
	//
	//	if err := stream.Stop(); err != nil {
	//		utils.Red.Print(err)
	//	}
}
