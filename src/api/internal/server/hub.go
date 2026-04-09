package server

import (
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type client struct {
	conn *websocket.Conn
	send chan []byte
}

// Hub manages the set of active WebSocket clients and fans out broadcasts.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*client]struct{}
	broadcast  chan []byte
	register   chan *client
	unregister chan *client
}

// NewHub creates an initialised Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*client]struct{}),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *client),
		unregister: make(chan *client),
	}
}

// Run processes register/unregister/broadcast events. Must be called in its own goroutine.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends data to all connected clients.
func (h *Hub) Broadcast(data []byte) {
	h.broadcast <- data
}
