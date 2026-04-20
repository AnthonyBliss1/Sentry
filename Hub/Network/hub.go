package network

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sync"

	utils "github.com/anthonybliss1/Sentry/Hub/Utils"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/hashicorp/mdns"
)

//go:embed templates/stream.html
var templateFS embed.FS

var watchTemplate = template.Must(template.ParseFS(templateFS, "templates/stream.html"))

const (
	RoomServiceLabel = "_Sentry-Hub-Room-Service._http"
)

type WatchPageData struct {
	Title       string
	WebRTCBase  string
	DefaultPath string
}

type StreamsResponse struct {
	Streams []string `json:"streams"`
}

type mediaMTXPathListResponse struct {
	Items []struct {
		Name string `json:"name"`
	} `json:"items"`
}

type Hub struct {
	Concierge
	rsMDNS *mdns.Server

	Hostname string
	LanIP    net.IP

	Mu sync.Mutex
}

// Server Control Functions
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (h *Hub) StartRoomService() {
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

func (h *Hub) WatchHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "cam"
	}

	data := WatchPageData{
		Title:       "Sentry Command Center",
		WebRTCBase:  h.WebRTCBase,
		DefaultPath: path,
	}

	if err := watchTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Hub) StreamsHandler(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://127.0.0.1:9997/v3/paths/list", nil)
	if err != nil {
		http.Error(w, "failed to build MediaMTX request", http.StatusInternalServerError)
		return
	}

	req.SetBasicAuth("sentryapi", "strongpassword")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "failed to query MediaMTX", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "MediaMTX returned non-200", http.StatusBadGateway)
		return
	}

	var apiResp mediaMTXPathListResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		http.Error(w, "failed to decode MediaMTX response", http.StatusInternalServerError)
		return
	}

	streams := make([]string, 0, len(apiResp.Items))
	for _, item := range apiResp.Items {
		if item.Name != "" {
			streams = append(streams, item.Name)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StreamsResponse{
		Streams: streams,
	})
}
