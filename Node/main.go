package main

import (
	"log"
	"sync"

	network "github.com/anthonybliss1/Sentry/Node/Network"
	utils "github.com/anthonybliss1/Sentry/Node/Utils"
)

var node network.NodeClient

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
	action := make(chan network.Message)

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
		if err := node.DialCommander(action); err != nil {
			log.Fatal(err)
		}
	}()

	// background task to respond to ws actions received
	utils.Blue.Println("> Deploying Stream Controller...")
	go func() {
		node.IsRunning = true
		if err := node.StreamController(action); err != nil {
			log.Panic(err)
		}
	}()

	// start streaming
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	utils.Blue.Println("> Publishing Video Stream...")
	go func() {
		if err := node.PublishStream(utils.Hostname); err != nil {
			log.Fatal(err)
		}
	}()

	select {}
}
