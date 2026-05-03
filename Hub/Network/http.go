package network

import (
	"encoding/json"
	"net/http"
)

type Concierge struct {
	RTSPPublishBase string `json:"rtsp_publish_base"`
	WebRTCBase      string `json:"webrtc_base"`
	HLSBase         string `json:"hls_base"`
}

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

	data := WatchPageData{
		Title:       "Sentry Command Center",
		WebRTCBase:  c.WebRTCBase,
		DefaultPath: path,
	}

	if err := watchTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
