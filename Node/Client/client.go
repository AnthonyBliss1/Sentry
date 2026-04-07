package client

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/hashicorp/mdns"
)

// service names to search for
const (
	WSService = "_Sentry-Hub-WS._tcp"
	FSService = "_Sentry-Hub-FS._tcp"
)

// stdout styling
var (
	green = color.New(color.FgGreen)
	blue  = color.New(color.FgBlue)
	red   = color.New(color.FgRed)
)

type Websocket struct {
	Server *http.Server

	URL      string
	Hostname string
	Addr     string
	Port     int
}

type FileServer struct {
	Server *http.Server

	URL      string
	Hostname string
	Addr     string
	Port     int
}

type HubServer struct {
	WS Websocket
	FS FileServer
}

type NodeClient struct {
	Mu  sync.Mutex
	Hub HubServer
}

func init() {
	// setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
}

func (n *NodeClient) lookupWS() error {
	entriesCH := make(chan *mdns.ServiceEntry, 16)

	go func() {
		for entry := range entriesCH {
			// frontline check
			if !strings.Contains(entry.Name, WSService) || entry.Port != 8000 {
				continue
			}

			// make sure there is a address
			if entry.AddrV4 == nil {
				continue
			}

			// slog.Debug("Hub Server found", "Found WS Entry", entry.Name, "Port", entry.Port)

			ws := Websocket{Hostname: entry.Host, Addr: entry.AddrV4.String(), Port: entry.Port}
			wsURL := fmt.Sprintf("ws://%s:%d/ws", ws.Addr, ws.Port)
			ws.URL = wsURL

			// first come first serve (for now, will change to hostname targeting i think later)
			n.Mu.Lock()
			if n.Hub.WS == (Websocket{}) {
				green.Print("> Stored WS Server ")
				blue.Printf("[ %s ]\n", n.Hub.WS.Hostname)
				n.Hub.WS = ws
			}
			n.Mu.Unlock()
		}
	}()

	err := mdns.Lookup(WSService, entriesCH)
	close(entriesCH)
	return err
}

func (n *NodeClient) lookupFS() error {
	entriesCH := make(chan *mdns.ServiceEntry, 16)

	go func() {
		for entry := range entriesCH {
			// frontline check
			if !strings.Contains(entry.Name, FSService) || entry.Port != 8080 {
				continue
			}

			// make sure there is a address
			if entry.AddrV4 == nil {
				continue
			}

			// slog.Debug("Hub Server found", "Found FS Entry", entry.Name, "Port", entry.Port)

			fs := FileServer{Hostname: entry.Host, Addr: entry.AddrV4.String(), Port: entry.Port}
			fsURL := fmt.Sprintf("http://%s:%d", fs.Addr, fs.Port)
			fs.URL = fsURL

			// first come first serve (for now, will change to hostname targeting i think later)
			n.Mu.Lock()
			if n.Hub.FS == (FileServer{}) {
				green.Print("> Stored FS Server ")
				blue.Printf("[ %s ]\n", n.Hub.FS.Hostname)
				n.Hub.FS = fs
			}
			n.Mu.Unlock()
		}
	}()

	err := mdns.Lookup(FSService, entriesCH)
	close(entriesCH)
	return err
}

// encapsulate both service lookup functions

func (n *NodeClient) MDNSLookup() error {
	if err := n.lookupWS(); err != nil {
		return err
	}

	if err := n.lookupFS(); err != nil {
		return err
	}

	return nil
}
