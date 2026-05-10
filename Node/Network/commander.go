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

func (c *Commander) DialCommander(action chan<- Message) error {
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

				// after confirming the recipient, send the msg to the channel
				// channel used to pass data to video publishing go routine
				utils.Blue.Println("> Sending to channel...")
				action <- *msg
			}
		}
	}
}
