package network

import (
	"fmt"
	"sync"
)

type DetectionEvent struct {
	Stream         string `json:"stream"`
	DogDetected    bool   `json:"dog_detected"`
	PersonDetected bool   `json:"person_detected"`
	Timestamp      string `json:"timestamp"`
}

type DetectionBroker struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}

	done chan struct{}
	once sync.Once
}

func CreateDetectionBroker() *DetectionBroker {
	return &DetectionBroker{
		clients: make(map[chan []byte]struct{}),
		done:    make(chan struct{}),
	}
}

func (b *DetectionBroker) Shutdown() {
	b.once.Do(func() {
		close(b.done)
	})
}

func (b *DetectionBroker) Broadcast(payload []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.clients {
		select {
		case ch <- payload:
		default:
			fmt.Println("[SSE dropping event... ]")
		}
	}
}

func (b *DetectionBroker) AddClient() chan []byte {
	ch := make(chan []byte, 16)

	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()

	return ch
}

func (b *DetectionBroker) RemoveClient(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}
