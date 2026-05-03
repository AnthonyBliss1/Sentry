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
	Recipient string `json:"recipient"`
	Action    string `json:"action"`

	ConnectedClients map[*Client]bool `json:"connected_clients"`
}

// Commander
// ~~~~~~~~~~~~~~

type Commander struct {
	Clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	shutdown   chan struct{}
}

func (cd *Commander) Broadcast(msg Message) {
	for client := range cd.Clients {
		select {
		case client.Send <- msg:
		default:
			close(client.Send)
			delete(cd.Clients, client)
		}
	}
}

func (cd *Commander) RunCommander() {
	for {
		select {
		case client := <-cd.register:
			cd.Clients[client] = true
			cd.Broadcast(Message{Action: "new-connection", ConnectedClients: cd.Clients})

		case client := <-cd.unregister:
			if _, ok := cd.Clients[client]; ok {
				delete(cd.Clients, client)
				close(client.Send)
				cd.Broadcast(Message{Action: "closed-connection", ConnectedClients: cd.Clients})
			}

		case msg := <-cd.broadcast:
			cd.Broadcast(msg)

		case <-cd.shutdown:
			cd.Broadcast(Message{Action: "server-shutdown"})
		}
	}
}

func (cd *Commander) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error upgrading Http to WS", "Error", err)
		return
	}
	client := &Client{Commander: *cd, Conn: conn, Send: make(chan Message, 256)}
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
	Commander

	Hostname string
	Conn     *websocket.Conn
	Send     chan Message
}

func (c *Client) readPump() {
	defer func() {
		c.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		var msg *Message

		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("WebSocket Error", "Error", err)
			}
			break
		}
		c.broadcast <- *msg
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
