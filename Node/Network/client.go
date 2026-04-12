package network

import (
	"fmt"
	"strings"
	"sync"

	utils "github.com/anthonybliss1/Sentry/Node/Utils"
	video "github.com/anthonybliss1/Sentry/Node/Video"
	"github.com/hashicorp/mdns"
)

// service names to search for
// TODO: Build out WS client
const (
	TCPService = "_Sentry-Hub-TCP._tcp"
)

type TCPServer struct {
	URL      string
	Hostname string
	Addr     string
	Port     int
}

type NodeClient struct {
	TCP    TCPServer
	Stream video.Stream

	Mu sync.Mutex
}

// MDNS Lookups
// ~~~~~~~~~~~~~~~~~~~~~~~

// keeping this wrapper function because i will add WS back

func (n *NodeClient) MDNSLookup() error {
	n.Mu.Lock()
	hasTCP := n.TCP != (TCPServer{})
	n.Mu.Unlock()

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	if !hasTCP {
		wg.Go(func() {
			if err := n.lookupTCP(); err != nil {
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

func (n *NodeClient) lookupTCP() error {
	entriesCH := make(chan *mdns.ServiceEntry, 16)
	errCH := make(chan error, 1)

	go func() {
		errCH <- mdns.Lookup(TCPService, entriesCH)
		close(entriesCH)
	}()

	for entry := range entriesCH {
		// frontline check
		if !strings.Contains(entry.Name, TCPService) || entry.Port != 9000 {
			continue
		}

		// make sure there is a address
		if entry.AddrV4 == nil {
			continue
		}

		tcp := TCPServer{Hostname: entry.Host, Addr: entry.AddrV4.String(), Port: entry.Port}
		tcp.URL = fmt.Sprintf("tcp://%s:%d", tcp.Addr, tcp.Port)

		// first come first serve (for now, will change to hostname targeting i think later)
		n.Mu.Lock()
		if n.TCP == (TCPServer{}) {
			utils.Blue.Printf("[ Stored TCP Server - %s ]\n", tcp.Hostname)
			n.TCP = tcp
		}
		n.Mu.Unlock()
	}

	return <-errCH
}
