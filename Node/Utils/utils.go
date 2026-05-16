package utils

import (
	"log"
	"os"

	"github.com/fatih/color"
)

var (
	Hostname string
	Green    = color.New(color.FgGreen) // debug
	Blue     = color.New(color.FgBlue)  // actions
	Red      = color.New(color.FgRed)   // warnings
)

func init() {
	var err error

	Hostname, err = os.Hostname()
	if err != nil {
		log.Fatalf("failed to collect hostname: %q", err)
	}
}
