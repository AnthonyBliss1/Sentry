package utils

import (
	"net"

	"github.com/fatih/color"
)

var (
	Green = color.New(color.FgGreen) // debug
	Blue  = color.New(color.FgBlue)  // actions
	Red   = color.New(color.FgRed)   // warnings
)

func LANIPv4() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}
