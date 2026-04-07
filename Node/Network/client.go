package network

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
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
	URL      string
	Hostname string
	Addr     string
	Port     int
}

type FileServer struct {
	Client *http.Client

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

// Init Logger
// ~~~~~~~~~~~~~~~~~~~~~~

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
}

// MDNS Lookups
// ~~~~~~~~~~~~~~~~~~~~~~~

func (n *NodeClient) MDNSLookup() error {
	if err := n.lookupWS(); err != nil {
		return err
	}

	if err := n.lookupFS(); err != nil {
		return err
	}

	return nil
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

// FS Upload
// ~~~~~~~~~~~~~~~~~~

func (n *NodeClient) UploadFile(filePath string) error {
	// ensure nodeclient has a valid address
	if n.Hub.FS.URL == "" {
		log.Fatal(errors.New("[FS] no adress found"))
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("[FS] file not found: %w", err)
		}
	}

	// safeguard against some potential file write issues
	if len(b) == 0 {
		return fmt.Errorf("[FS] read empty file: %w", err)
	}

	req, err := http.NewRequest("Port", n.Hub.FS.URL+"/upload/123", bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("[FS] failed to create http request: %w", err)
	}

	// shouldnt create a client the every time
	var client *http.Client

	n.Mu.Lock()
	if n.Hub.FS.Client == nil {
		// create and store client
		client = &http.Client{}
		n.Hub.FS.Client = client
	} else {
		// grab stored client
		client = n.Hub.FS.Client
	}
	n.Mu.Unlock()

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("[FS] failed to do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[FS] not ok response status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("[FS] failed to read response body: %w", err)
	}
	defer resp.Body.Close()

	green.Printf("[FS] %s\n", string(bodyBytes))

	return nil
}
