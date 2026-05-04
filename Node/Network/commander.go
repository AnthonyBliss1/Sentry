package network

import (
	"encoding/json"
	"fmt"
	"os"

	utils "github.com/anthonybliss1/Sentry/Node/Utils"
	"github.com/gorilla/websocket"
)

type Commander struct {
	CommanderServiceURL string
	Conn                *websocket.Conn
}

type Message struct {
	Recipient string `json:"recipient"` // this will be the hostname for the intended node
	Action    string `json:"action"`
}

func (m *Message) String() string {
	b, _ := json.MarshalIndent(m, "", "  ")
	return string(b)
}

func (c *Commander) DialCommander() error {
	var err error
	hn, _ := os.Hostname()

	c.Conn, _, err = websocket.DefaultDialer.Dial(c.CommanderServiceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to dail commander: %w", err)
	}
	defer c.Conn.Close()

	// continue listening
	for {
		var msg *Message

		if err := c.Conn.ReadJSON(&msg); err != nil {
			utils.Red.Printf("> Failed to read message from commander: %v\n", err)
		} else {
			// node will determine if the message is intended for itself
			if msg.Recipient == hn {
				utils.Blue.Println("WS Message Received: ", msg)

				// depending on action, should then send msg to channel to communicate with the publish stream go routine
				// should turn the publish stream go routine into a stream controller than will start and stop depending on Actions
			}
		}
	}
}
