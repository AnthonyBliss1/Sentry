package main

import (
	"log"
	"time"

	client "github.com/anthonybliss1/Sentry/Node/Client"
	video "github.com/anthonybliss1/Sentry/Node/Video"
)

func main() {
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

	// start recording and creating segments
	stream, err := video.StartStream()
	if err != nil {
		log.Print(err)
	}

	time.Sleep(5 * time.Second)

	if err := stream.Stop(); err != nil {
		log.Print(err)
	}
}
