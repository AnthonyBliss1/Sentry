package network

import (
	"embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"

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

	HTTPMDNS *mdns.Server
	WSMDNS   *mdns.Server

	HTTPSrv *http.Server

	Hostname string
	LanIP    net.IP
}

// Server Control Functions
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (h *Hub) StartConciergeService() {
	if h.Detections == nil {
		h.Detections = CreateDetectionBroker()
	}

	r := chi.NewRouter()

	r.Use(middleware.Logger)

	fs := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	r.Get("/room-service", h.RoomServiceHandler)
	r.Get("/watch", h.WatchHandler)
	r.Get("/api/streams", h.StreamsHandler)

	r.Post("/api/detections", h.PublishDetectionHandler)
	r.Get("/api/detections/events", h.DetectionEventHandler)

	addr := ":8000"

	h.HTTPSrv = &http.Server{Addr: addr, Handler: r}

	go func() {
		if err := h.HTTPSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Red.Printf("Concierge Service error: %q\n", err)
		}
	}()

	utils.Green.Printf("[ Concierge Service listening on http://%s:%d ]\n", h.LanIP.String(), 8000)
}

func (h *Hub) StartCommanderService() {
	h.Commander = NewCommander(h.SetAlias)

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
		if err := ws.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Red.Printf("Commander Service error: %q\n", err)
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

	h.HTTPMDNS, err = mdns.NewServer(&mdns.Config{Zone: httpService})
	if err != nil {
		utils.Red.Printf("HTTP MDNS Server Shutdown: %q\n", err)
		return
	}

	h.WSMDNS, err = mdns.NewServer(&mdns.Config{Zone: wsService})
	if err != nil {
		utils.Red.Printf("WebSocket MDNS Server Shutdown: %q\n", err)
		return
	}
}

func (h *Hub) SetAlias(hostname string, alias string) {
	h.aliasMu.Lock()
	defer h.aliasMu.Unlock()

	if h.aliases == nil {
		h.aliases = make(map[string]string)
	}

	hostname = strings.TrimSpace(hostname)
	alias = strings.TrimSpace(alias)

	// dont allow empty values
	if hostname == "" || alias == "" {
		return
	}

	h.aliases[hostname] = alias
}
