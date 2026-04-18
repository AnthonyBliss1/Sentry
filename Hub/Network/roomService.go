package network

import (
	"encoding/json"
	"net/http"
)

type Concierge struct {
	RoomURL   string `json:"url"`
	APIKey    string `json:"api-key"`
	APISecret string `json:"api-secret"`
	RoomName  string `json:"room-name"`
}

func (c *Concierge) RoomServiceHandler(w http.ResponseWriter, r *http.Request) {
	// quick check
	if c.RoomURL == "" {
		http.Error(w, "concierge not set - empty url", http.StatusInternalServerError)
		return
	}

	// struct -> bytes
	data, err := json.Marshal(c)
	if err != nil {
		http.Error(w, "failed to collect room information", http.StatusInternalServerError)
	}

	// just sharing the room information via concierge
	w.Write(data)
}
