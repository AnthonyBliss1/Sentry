package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"sync"

	utils "github.com/anthonybliss1/Sentry/Hub/Utils"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hashicorp/mdns"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

const (
	RoomServiceLabel = "_Sentry-Hub-Room-Service._http"
)

type Hub struct {
	Concierge
	Room   *livekit.Room
	rsMDNS *mdns.Server

	Hostname string
	LanIP    net.IP

	LKServerCMD *exec.Cmd
	Mu          sync.Mutex
}

// Server Control Functions
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (h *Hub) StartLKServer() error {
	cmd := exec.Command("livekit-server",
		"--dev",
		"--bind", "0.0.0.0",
	)

	cmd.Stderr = io.Discard
	cmd.Stdout = io.Discard

	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}

func (h *Hub) SetRoom(roomName string, url string) (err error) {
	h.Concierge = Concierge{RoomName: roomName, RoomURL: url, APIKey: "devkey", APISecret: "secret"}

	roomClient := lksdk.NewRoomServiceClient(h.RoomURL, h.APIKey, h.APISecret)

	h.Room, err = roomClient.CreateRoom(context.Background(), &livekit.CreateRoomRequest{Name: h.RoomName})
	if err != nil {
		return fmt.Errorf("failed to create livekit room: %w", err)
	}

	return nil
}

func (h *Hub) StartRoomService() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/room-service", h.RoomServiceHandler)
	addr := fmt.Sprintf("%s:%d", h.LanIP.String(), 8000)

	go func() {
		if err := http.ListenAndServe(addr, r); err != nil {
			utils.Red.Printf("RS Server Shutdown: %q\n", err)
		}
	}()

	utils.Green.Printf("[ Room Service listening on :%d ]\n", 8000)
}

func (h *Hub) StartMDNS() {
	var err error

	info := []string{"Sentry Hub"}

	tcpService, _ := mdns.NewMDNSService(h.Hostname, RoomServiceLabel, "", "", 8000, []net.IP{h.LanIP}, info)

	utils.Green.Println("[ MDNS Server advertising Room Service on :8000 ]")
	h.rsMDNS, err = mdns.NewServer(&mdns.Config{Zone: tcpService})
	if err != nil {
		utils.Red.Printf("TCP MDNS Server Shutdown: %q\n", err)
		return
	}
}
