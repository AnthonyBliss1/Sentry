package network

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Message struct {
	Recipient      string `json:"recipient"` // this will be the hostname for the intended node
	Action         string `json:"action"`
	NewNode        Node   `json:"new_node"`
	ConnectedNodes []Node `json:"connected_nodes"` // for the frontend
}

type InboundMessage struct {
	Client  *Client
	Message Message
}

// Commander
// ~~~~~~~~~~~~~~

type Node struct {
	Hostname string `json:"hostname"`
}

type Commander struct {
	Clients        map[*Client]bool
	ConnectedNodes map[string]Node
	broadcast      chan InboundMessage
	register       chan *Client
	unregister     chan *Client
	shutdown       chan struct{}
}

func NewCommander() *Commander {
	return &Commander{
		Clients:        make(map[*Client]bool),
		ConnectedNodes: make(map[string]Node),
		broadcast:      make(chan InboundMessage),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		shutdown:       make(chan struct{}),
	}
}

func (cd *Commander) connectedNodeList() []Node {
	nodes := make([]Node, 0, len(cd.ConnectedNodes))
	for _, node := range cd.ConnectedNodes {
		nodes = append(nodes, node)
	}
	return nodes
}

func (cd *Commander) Broadcast(msg Message) {
	for client := range cd.Clients {
		select {
		case client.Send <- msg: // send msg
		default: // if closed channel
			close(client.Send)
			delete(cd.Clients, client)

			// remove if connected node
			if client.isNode && client.Hostname != "" {
				delete(cd.ConnectedNodes, client.Hostname)
			}
		}
	}
}

func (cd *Commander) RunCommander() {
	for {
		select {
		case client := <-cd.register:
			cd.Clients[client] = true

			// share the client list on new client register (for frontend client connecting when nodes are already connected)
			cd.Broadcast(Message{Recipient: "All", Action: "new_connection", ConnectedNodes: cd.connectedNodeList()})

		case client := <-cd.unregister:
			if _, ok := cd.Clients[client]; ok {
				delete(cd.Clients, client)
				close(client.Send)
			}

			// unregister a connected node
			if client.isNode && client.Hostname != "" {
				delete(cd.ConnectedNodes, client.Hostname)

				cd.Broadcast(Message{Recipient: "All", Action: "node_disconnected", ConnectedNodes: cd.connectedNodeList()})
			}

		case inboundMsg := <-cd.broadcast:
			client := inboundMsg.Client
			msg := inboundMsg.Message

			switch msg.Action {
			case "register_node":
				hostname := msg.NewNode.Hostname

				client.isNode = true
				client.Hostname = hostname

				cd.ConnectedNodes[hostname] = msg.NewNode

				cd.Broadcast(Message{Recipient: "All", Action: "node_connected", ConnectedNodes: cd.connectedNodeList()})

			default:
				cd.Broadcast(msg)
			}

		case <-cd.shutdown:
			cd.Broadcast(Message{Action: "server-shutdown"})
			return
		}
	}
}

func (cd *Commander) WSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error upgrading Http to WS", "Error", err)
		return
	}
	client := &Client{Commander: cd, Conn: conn, Send: make(chan Message, 256)}
	client.register <- client

	go client.writePump()
	go client.readPump()
}

// Client
// ~~~~~~~~~~~~~~

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Client struct {
	*Commander
	Conn *websocket.Conn
	Send chan Message

	isNode   bool
	Hostname string
}

func (c *Client) readPump() {
	defer func() {
		c.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		var msg Message

		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("WebSocket Error", "Error", err)
			}
			break
		}
		c.broadcast <- InboundMessage{Client: c, Message: msg}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				return
			}

			c.Conn.WriteJSON(&msg)

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
