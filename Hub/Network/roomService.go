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
