package network

import (
	"embed"
	"fmt"
	"html/template"
	"net"
	"net/http"

	utils "github.com/anthonybliss1/Sentry/Hub/Utils"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hashicorp/mdns"
)

//go:embed templates/watch.html
var templateFS embed.FS

var watchTemplate = template.Must(template.ParseFS(templateFS, "templates/watch.html"))

const (
	ConciergeServiceLabel = "_Sentry-Hub-Concierge-Service._http"
	CommanderServiceLabel = "_Sentry-Hub-Commander-Service._ws"
)

type Hub struct {
	Concierge
	*Commander

	httpMDNS *mdns.Server
	wsMDNS   *mdns.Server

	Hostname string
	LanIP    net.IP
}

// Server Control Functions
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (h *Hub) StartConciergeService() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	fs := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	r.Get("/room-service", h.RoomServiceHandler)
	r.Get("/watch", h.WatchHandler)
	r.Get("/api/streams", h.StreamsHandler)

	addr := fmt.Sprintf("%s:%d", h.LanIP.String(), 8000)

	go func() {
		if err := http.ListenAndServe(addr, r); err != nil {
			utils.Red.Printf("HTTP Server Shutdown: %q\n", err)
		}
	}()

	utils.Green.Printf("[ Concierge Service listening on :%d ]\n", 8000)
}

func (h *Hub) StartCommanderService() {
	h.Commander = NewCommander()

	go h.RunCommander()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.WSHandler)

	addr := fmt.Sprintf("%s:%d", h.LanIP.String(), 9000)
	h.wsURL = fmt.Sprintf("ws://%s/ws", addr)

	ws := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := ws.ListenAndServe(); err != nil {
			utils.Red.Printf("WebSocket Server Shutdown: %q\n", err)
		}
	}()

	utils.Green.Printf("[ Commander Service listening on :%d ]\n", 9000)
}

func (h *Hub) StartMDNS() {
	var err error

	info := []string{"Sentry Hub"}

	httpService, _ := mdns.NewMDNSService(h.Hostname, ConciergeServiceLabel, "", "", 8000, []net.IP{h.LanIP}, info)
	wsService, _ := mdns.NewMDNSService(h.Hostname, CommanderServiceLabel, "", "", 9000, []net.IP{h.LanIP}, info)

	utils.Green.Println("[ MDNS Server advertising Concierge Service on :8000 ]")
	utils.Green.Println("[ MDNS Server advertising Commander Service on :9000 ]")

	h.httpMDNS, err = mdns.NewServer(&mdns.Config{Zone: httpService})
	if err != nil {
		utils.Red.Printf("HTTP MDNS Server Shutdown: %q\n", err)
		return
	}

	h.wsMDNS, err = mdns.NewServer(&mdns.Config{Zone: wsService})
	if err != nil {
		utils.Red.Printf("WebSocket MDNS Server Shutdown: %q\n", err)
		return
	}
}
