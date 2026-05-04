package main

import (
	"log"
	"os"
	"sync"

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

	// find Hub services
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	var wg sync.WaitGroup

	utils.Blue.Println("> Looking for Concierge Service...")
	utils.Blue.Println("> Looking for Commander Service...")
	wg.Go(node.ConciergeServiceLookup)
	wg.Go(node.CommanderServiceLookup)

	wg.Wait()

	utils.Blue.Println("> Fetching Concierge...")
	if err := node.FetchConcierge(); err != nil {
		log.Fatal(err)
	}

	// background task to continue listening to ws
	utils.Blue.Println("> Dialing Commander...")
	go func() {
		if err := node.DialCommander(); err != nil {
			log.Fatal(err)
		}
	}()

	deviceID, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}

	// start recording and creating segments
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	utils.Blue.Println("> Publishing Video Stream...")
	go func() {
		_, err = node.PublishStream(deviceID)
		if err != nil {
			log.Fatal(err)
		}
	}()

	select {} // block forever

	// test end
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	//	time.Sleep(120 * time.Second)
	//
	//	if err := stream.Stop(); err != nil {
	//		utils.Red.Print(err)
	//	}
}
