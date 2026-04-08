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
	WS Websocket
	FS FileServer

	wsMDNS *mdns.Server
	fsMDNS *mdns.Server

	LanIP    net.IP
	Hostname string
	Mu       sync.Mutex
}

// Server Control Functions
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (h *Hub) StartWS() {
	// need to implement the Gorilla hub model

	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// ServeWS(p.Hub, w, r)
	})

	h.WS = Websocket{&http.Server{Addr: h.LanIP.String() + ":8000", Handler: mux}}

	go func() {
		if err := h.WS.ListenAndServe(); err != nil {
			utils.Red.Printf("Server Shutdown: %q\n", err)
		}
	}()

	utils.Green.Println("[ Websocket Server listening on :8000 ]")
}

func (h *Hub) StartFS() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Post("/upload/{deviceID}/{fileName}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "deviceID") // grab the id from the qParam
		fileName := chi.URLParam(r, "fileName")

		msg := fmt.Sprintf("Successfully Uploaded File [%s] for device: %s", fileName, id) // test msg

		w.Write([]byte(msg)) // write msg
	})

	// will probably allow the user to configure the port? not sure
	// or use some obscure port
	addr := fmt.Sprintf("%s:%d", h.LanIP.String(), 8080)

	h.FS = FileServer{&http.Server{Addr: addr, Handler: r}}

	go func() {
		if err := h.FS.ListenAndServe(); err != nil {
			utils.Red.Printf("Server Shutdown: %q\n", err)
		}
	}()

	utils.Green.Println("[ File Server listening on :8080 ]")
}

func (h *Hub) StartMDNS() {
	var err error

	info := []string{"Sentry Hub"}

	wsService, _ := mdns.NewMDNSService(h.Hostname, "_Sentry-Hub-WS._tcp", "", "", 8000, []net.IP{h.LanIP}, info)
	fsService, _ := mdns.NewMDNSService(h.Hostname, "_Sentry-Hub-FS._tcp", "", "", 8080, []net.IP{h.LanIP}, info)

	utils.Green.Println("[ MDNS Server advertising WSService on :8000 ]")
	h.wsMDNS, err = mdns.NewServer(&mdns.Config{Zone: wsService})
	if err != nil {
		utils.Red.Printf("WS MDNS Server Shutdown: %q\n", err)
	}

	utils.Green.Println("[ MDNS Server advertising FSService on :8080 ]")
	h.fsMDNS, err = mdns.NewServer(&mdns.Config{Zone: fsService})
	if err != nil {
		utils.Red.Printf("FS MDNS Server Shutdown: %q\n", err)
	}
}
