package network

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

type NodeClient struct {
	WS Websocket
	FS FileServer

	Mu sync.Mutex
}

// MDNS Lookups
// ~~~~~~~~~~~~~~~~~~~~~~~

func (n *NodeClient) MDNSLookup() error {
	n.Mu.Lock()
	hasWS := n.WS != (Websocket{})
	hasFS := n.FS != (FileServer{})
	n.Mu.Unlock()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	if !hasWS {
		wg.Go(func() {
			if err := n.lookupWS(); err != nil {
				errCh <- err
			}
		})
	}

	if !hasFS {
		wg.Go(func() {
			if err := n.lookupFS(); err != nil {
				errCh <- err
			}
		})
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *NodeClient) lookupWS() error {
	entriesCH := make(chan *mdns.ServiceEntry, 16)
	errCH := make(chan error, 1)

	go func() {
		errCH <- mdns.Lookup(WSService, entriesCH)
		close(entriesCH)
	}()

	for entry := range entriesCH {
		// frontline check
		if !strings.Contains(entry.Name, WSService) || entry.Port != 8000 {
			continue
		}

		// make sure there is a address
		if entry.AddrV4 == nil {
			continue
		}

		ws := Websocket{Hostname: entry.Host, Addr: entry.AddrV4.String(), Port: entry.Port}
		ws.URL = fmt.Sprintf("ws://%s:%d/ws", ws.Addr, ws.Port)

		// first come first serve (for now, will change to hostname targeting i think later)
		n.Mu.Lock()
		if n.WS == (Websocket{}) {
			blue.Printf("[ Stored WS Server - %s ]\n", ws.Hostname)
			n.WS = ws
		}
		n.Mu.Unlock()
	}

	return <-errCH
}

func (n *NodeClient) lookupFS() error {
	entriesCH := make(chan *mdns.ServiceEntry, 16)
	errCH := make(chan error, 1)

	go func() {
		errCH <- mdns.Lookup(FSService, entriesCH)
		close(entriesCH)
	}()

	for entry := range entriesCH {
		// frontline check
		if !strings.Contains(entry.Name, FSService) || entry.Port != 8080 {
			continue
		}

		// make sure there is a address
		if entry.AddrV4 == nil {
			continue
		}

		fs := FileServer{Hostname: entry.Host, Addr: entry.AddrV4.String(), Port: entry.Port}
		fs.URL = fmt.Sprintf("http://%s:%d", fs.Addr, fs.Port)

		// first come first serve (for now, will change to hostname targeting i think later)
		n.Mu.Lock()
		if n.FS == (FileServer{}) {
			blue.Printf("[ Stored FS Server - %s ]\n", fs.Hostname)
			n.FS = fs
		}
		n.Mu.Unlock()
	}

	return <-errCH
}

// FS Upload
// ~~~~~~~~~~~~~~~~~~

func (n *NodeClient) UploadFile(filePath string) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("[FS] failed to read file: %w", err)
	}

	// safeguard against some potential file write issues
	if len(b) == 0 {
		return fmt.Errorf("[FS] read empty file: %s", filepath.Base(filePath))
	}

	n.Mu.Lock()

	// ensure nodeclient has a valid address
	if n.FS.URL == "" {
		n.Mu.Unlock()
		return errors.New("[FS] no url found")
	}

	// need to swap out 123 for unique id maybe
	url := n.FS.URL + "/upload/123/" + filepath.Base(filePath)

	// ensure nodeclient has a valid client
	if n.FS.Client == nil {
		n.Mu.Unlock()
		return errors.New("[FS] no http client found")
	}

	client := n.FS.Client
	n.Mu.Unlock()

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("[FS] failed to create http request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("[FS] failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[FS] not ok response status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("[FS] failed to read response body: %w", err)
	}

	green.Printf("[FS] %s\n", string(bodyBytes))

	return nil
}
