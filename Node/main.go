package main

import (
	"flag"
	"time"

	deploy "github.com/anthonybliss1/Sentry/Node/Deploy"
	network "github.com/anthonybliss1/Sentry/Node/Network"
	utils "github.com/anthonybliss1/Sentry/Node/Utils"
)

func main() {
	// TODO: implement these checks
	//
	// should run checks for camera setup
	//
	// if err := InitCamera(); err != nil {
	// log.Fatal(err)
	// }

	d := flag.Bool("deploy", false, "creates systemD file for the sentry-node program")

	flag.Parse()

	if *d {
		if err := deploy.DeployNode(); err != nil {
			utils.Red.Printf("failed to deploy node: %q\n", err)
			return
		}

		utils.Blue.Println("\nSentry Node successfully deployed!")
		return
	}

	for {
		if err := utils.SetHostname(); err != nil {
			utils.Red.Println(err)
		}

		if err := runNodeSession(); err != nil {
			utils.Red.Printf("> Node session ended: %v\n", err)
		}

		utils.Blue.Println("> Restarting hub lookup / websocket / publisher...")
		time.Sleep(2 * time.Second)
	}
}

func runNodeSession() error {
	node := &network.NodeClient{}

	if err := node.HubLookup(); err != nil {
		return err
	}

	action := make(chan network.Message, 8)
	controllerDone := make(chan error, 1)

	utils.Blue.Println("> Deploying Stream Controller...")
	go func() {
		controllerDone <- node.StreamController(action)
	}()

	// start stream via StreamController
	action <- network.Message{Recipient: utils.Hostname, Action: "start"}

	// blocks until websocket dies
	err := node.DialCommander(action)
	utils.Red.Printf("> commander disconnected: %v\n", err)

	// stop stream via StreamController
	action <- network.Message{Recipient: utils.Hostname, Action: "stop"}
	close(action)

	select {
	case controllerErr := <-controllerDone:
		if controllerErr != nil {
			utils.Red.Printf("> stream controller stopped with error: %v\n", controllerErr)
		}
	case <-time.After(5 * time.Second):
		utils.Red.Println("> timed out waiting for stream controller shutdown")
	}

	return err
}
