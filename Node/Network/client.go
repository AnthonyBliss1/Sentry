package network

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	utils "github.com/anthonybliss1/Sentry/Node/Utils"
	"github.com/hashicorp/mdns"
)

// service names to search for
const (
	RoomServiceLabel = "_Sentry-Hub-Room-Service._http"
)

// Room Service API discovered from MDNS

type NodeClient struct {
	Concierge
	RoomServiceAPI string
	Stream         Stream

	Mu sync.Mutex
}

// MDNS Lookups
// ~~~~~~~~~~~~~~~~~~~~~~~

func (n *NodeClient) RoomServiceLookup() {
	for {
		entriesCH := make(chan *mdns.ServiceEntry, 16)

		mdns.Lookup(RoomServiceLabel, entriesCH)
		close(entriesCH)

		for entry := range entriesCH {
			// frontline check
			if !strings.Contains(entry.Name, RoomServiceLabel) || entry.Port != 8000 {
				continue
			}

			// make sure there is a address
			if entry.AddrV4 == nil {
				continue
			}

			// build api url
			url := fmt.Sprintf("http://%s:%d/room-service", entry.AddrV4.String(), entry.Port)

			// first come first serve (for now, will change to hostname targeting i think later)
			n.Mu.Lock()
			if n.RoomServiceAPI == "" {
				utils.Green.Printf("[ Stored API URL from <- %s ]\n", entry.Host)
				n.RoomServiceAPI = url
				n.Mu.Unlock()
				return
			}
			n.Mu.Unlock()
		}

		time.Sleep(2 * time.Second)
	}
}

func (n *NodeClient) FetchConcierge() error {
	if n.RoomServiceAPI == "" {
		return errors.New("no roomservice url set, cannot fetch concierge")
	}

	resp, err := http.Get(n.RoomServiceAPI)
	if err != nil {
		return fmt.Errorf("failed to fetch concierge: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read roomservice response: %w", err)
	}

	if err := json.Unmarshal(bodyBytes, &n.Concierge); err != nil {
		return fmt.Errorf("failed to unmarshal concierge bytes: %w", err)
	}

	if n.Concierge != (Concierge{}) {
		utils.Green.Println("[ Concierge Collected ]")
		utils.Green.Println(n) // this will print out the n.Concierge data (will change because its confusing)
	}

	return nil
}
