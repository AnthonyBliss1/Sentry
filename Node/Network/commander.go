package network

import (
	"encoding/json"
	"fmt"

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
	NewNode   Node   `json:"new_node"`
}

type Node struct {
	Hostname string `json:"hostname"`
}

func (m *Message) String() string {
	b, _ := json.MarshalIndent(m, "", "  ")
	return string(b)
}

func (c *Commander) DialCommander(action chan<- Message) error {
	var err error

	c.Conn, _, err = websocket.DefaultDialer.Dial(c.CommanderServiceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to dail commander: %w", err)
	}
	defer c.Conn.Close()

	// send register_node message
	if err := c.RegisterNode(); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	// continue listening
	for {
		var msg *Message

		if err := c.Conn.ReadJSON(&msg); err != nil {
			return fmt.Errorf("commander websocket read failed: %w", err)
		} else {
			// node will determine if the message is intended for itself
			if msg.Recipient == utils.Hostname {
				utils.Green.Println("WS Message Received: ", msg)

				// after confirming the recipient, send the msg to the channel
				// channel used to pass data to video publishing go routine
				utils.Blue.Println("> Sending to channel...")
				action <- *msg
			}
		}
	}
}

func (c *Commander) RegisterNode() error {
	msg := Message{Recipient: "All", Action: "register_node", NewNode: Node{Hostname: utils.Hostname}}

	return c.Conn.WriteJSON(msg)
}
