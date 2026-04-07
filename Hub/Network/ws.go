package network

import (
	"net/http"
)

type Websocket struct {
	*http.Server
}

type Bus struct{}

type Message struct{}

// Websocket handlers here
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~
