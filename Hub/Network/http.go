package network

import (
	"encoding/json"
	"fmt"
	"html/template"
	"maps"
	"net/http"
	"sync"
	"time"
)

type Concierge struct {
	RTSPPublishBase string `json:"rtsp_publish_base"`
	WebRTCBase      string `json:"webrtc_base"`
	HLSBase         string `json:"hls_base"`
	Detections      *DetectionBroker

	aliases map[string]string
	aliasMu sync.Mutex

	wsURL string // private for watch template
}

type WatchPageData struct {
	Title              string
	WebRTCBase         string
	DefaultPath        string
	WebSocketURL       string
	DetectionEventsURL string

	Aliases   map[string]string // hostname is the key, alias is the value
	AliasJSON template.JS
}

type StreamsResponse struct {
	Streams []string `json:"streams"`
}

type mediaMTXPathListResponse struct {
	Items []struct {
		Name string `json:"name"`
	} `json:"items"`
}

func (c *Concierge) RoomServiceHandler(w http.ResponseWriter, r *http.Request) {
	// quick check
	if c.RTSPPublishBase == "" || c.WebRTCBase == "" || c.HLSBase == "" {
		http.Error(w, "concierge not set - empty field", http.StatusInternalServerError)
		return
	}

	// struct -> bytes
	data, err := json.Marshal(c)
	if err != nil {
		http.Error(w, "failed to collect room information", http.StatusInternalServerError)
	}

	// just sharing the room information via concierge
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (c *Concierge) WatchHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "cam"
	}

	// raw go maps dont work nicely with JS

	aliases := c.AliasSnapshot()

	aliasesJSON, err := json.Marshal(aliases)
	if err != nil {
		http.Error(w, "failed to marshal aliases", http.StatusInternalServerError)
		return
	}

	data := WatchPageData{
		Title:              "Sentry Command Center",
		WebRTCBase:         c.WebRTCBase,
		DefaultPath:        path,
		WebSocketURL:       c.wsURL,
		DetectionEventsURL: "/api/detections/events",
		Aliases:            aliases,
		AliasJSON:          template.JS(aliasesJSON),
	}

	if err := watchTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// wrapper for a mu lock to prevent races

func (c *Concierge) AliasSnapshot() map[string]string {
	c.aliasMu.Lock()
	defer c.aliasMu.Unlock()

	out := make(map[string]string, len(c.aliases))

	maps.Copy(out, c.aliases)

	return out
}

func (c *Concierge) StreamsHandler(w http.ResponseWriter, r *http.Request) {
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

func (c *Concierge) PublishDetectionHandler(w http.ResponseWriter, r *http.Request) {
	if c.Detections == nil {
		http.Error(w, "detection broker not configured", http.StatusInternalServerError)
		return
	}

	defer r.Body.Close()

	var event DetectionEvent
	// could probably switch to json unmarshall and bodybytes read?
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "invalid detection payload", http.StatusBadRequest)
		return
	}

	// handle missing data
	if event.Stream == "" {
		http.Error(w, "missing stream", http.StatusBadRequest)
		return
	}

	if !event.DogDetected && !event.PersonDetected {
		http.Error(w, "missing detection type", http.StatusBadRequest)
		return
	}

	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339Nano) // could be local time stamp not UTC
	}

	payload, err := json.Marshal(event)
	if err != nil {
		http.Error(w, "failed to encode detection event", http.StatusInternalServerError)
		return
	}

	c.Detections.Broadcast(payload)
	w.WriteHeader(http.StatusAccepted)
}

func (c *Concierge) DetectionEventHandler(w http.ResponseWriter, r *http.Request) {
	if c.Detections == nil {
		http.Error(w, "detection broker not configured", http.StatusInternalServerError)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := c.Detections.AddClient()
	defer c.Detections.RemoveClient(ch)

	fmt.Fprint(w, ": connected\n")
	fmt.Fprint(w, "retry: 3000\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case <-c.Detections.done:
			return

		case payload, ok := <-ch:
			if !ok {
				return
			}

			fmt.Fprint(w, "event: object_detection\n")
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()

		case <-heartbeat.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
