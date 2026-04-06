package main

import (
	"log"
	"time"

	video "github.com/anthonybliss1/Sentry/Node/Video"
)

func main() {
	stream, err := video.StartStream()
	if err != nil {
		log.Print(err)
	}

	time.Sleep(5 * time.Second)

	if err := stream.Stop(); err != nil {
		log.Print(err)
	}

	time.Sleep(2 * time.Second)

	stream, err = video.StartStream()
	if err != nil {
		log.Print(err)
	}

	time.Sleep(5 * time.Second)

	if err := stream.Stop(); err != nil {
		log.Print(err)
	}
}
