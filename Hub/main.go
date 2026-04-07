package main

import (
	network "github.com/anthonybliss1/Sentry/Hub/Network"
	"github.com/fatih/color"
)

var green = color.New(color.FgGreen)

func main() {
	green.Println("> Starting FS server on Port 8080...")
	network.CreateFS()
}
