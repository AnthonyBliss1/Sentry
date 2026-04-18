package network

import (
	"encoding/json"
	"fmt"

	lksdk "github.com/livekit/server-sdk-go"
)

// LiveKit Server credentials received from API

type Concierge struct {
	RoomURL   string `json:"url"`
	APIKey    string `json:"api-key"`
	APISecret string `json:"api-secret"`
	RoomName  string `json:"room-name"`

	room *lksdk.Room
}

func (c *Concierge) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")

	return string(b)
}

func (c *Concierge) JoinRoom() (err error) {
	c.room, err = lksdk.ConnectToRoom(c.RoomURL, lksdk.ConnectInfo{
		APIKey:              c.APIKey,
		APISecret:           c.APISecret,
		RoomName:            c.RoomName,
		ParticipantIdentity: "Gabagul",
	}, nil) // no callback since the node is only a publisher, dont need to sub to participant's tracks when joining the room
	if err != nil {
		return fmt.Errorf("failed to join room: %w", err)
	}

	return nil
}

func (c *Concierge) LeaveRoom() (ok bool) {
	if c.room == nil {
		return false
	}

	c.room.Disconnect()
	return true
}
