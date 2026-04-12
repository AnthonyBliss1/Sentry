package network

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	utils "github.com/anthonybliss1/Sentry/Hub/Utils"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hashicorp/mdns"
)

type Hub struct {
	TCP TCPServer

	fsMDNS *mdns.Server

	LanIP    net.IP
	Hostname string
	HLSDir   string
	Mu       sync.Mutex
}

// Server Control Functions
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (h *Hub) StartFS(hlsDir string) {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	// handle serving template and hls files on disk
	r.Get("/stream", StreamHandler(hlsDir))
	r.Handle("/hls/*", HLSFileServer(hlsDir))

	// will probably allow the user to configure the port? not sure
	// or use some obscure port
	addr := fmt.Sprintf("%s:%d", h.LanIP.String(), 8080)

	go func() {
		if err := http.ListenAndServe(addr, r); err != nil {
			utils.Red.Printf("Server Shutdown: %q\n", err)
		}
	}()

	utils.Green.Println("[ File Server listening on :8080 ]")
}

func (h *Hub) StartTCP(hlsDir string) {
	addr := fmt.Sprintf("%s:%d", h.LanIP.String(), 9000)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		utils.Red.Printf("TCP ingest listen failed: %v\n", err)
		return
	}

	h.TCP = TCPServer{listener: ln}

	utils.Green.Println("[ TCP Ingest listening on :9000 ]")

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				utils.Red.Printf("TCP accept error: %v\n", err)
				continue
			}

			h.TCP.wg.Add(1)
			go func() {
				defer h.TCP.wg.Done()
				handleTCPStream(conn, hlsDir)
			}()
		}
	}()
}

func (h *Hub) StartMDNS() {
	var err error

	info := []string{"Sentry Hub"}

	tcpService, _ := mdns.NewMDNSService(h.Hostname, "_Sentry-Hub-TCP._tcp", "", "", 9000, []net.IP{h.LanIP}, info)

	utils.Green.Println("[ MDNS Server advertising TCPService on :9000 ]")
	h.fsMDNS, err = mdns.NewServer(&mdns.Config{Zone: tcpService})
	if err != nil {
		utils.Red.Printf("TCP MDNS Server Shutdown: %q\n", err)
		return
	}
}
