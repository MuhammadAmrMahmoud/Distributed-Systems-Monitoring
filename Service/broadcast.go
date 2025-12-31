package service

import (
	"Distributed-Health-Monitoring/Repository"
	"Distributed-Health-Monitoring/models"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var GlobalHub *Hub

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

type ServiceStateChangeEvent struct {
	Type      string    `json:"type"` // service_state_change
	ServiceID uint      `json:"service_id"`
	Name      string    `json:"name"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Timestamp time.Time `json:"timestamp"`
}
func BroadcastStateChange(
	service models.ExternalService,
	change *Repository.StateChange,
) {
	event := ServiceStateChangeEvent{
		Type:      "service_state_change",
		ServiceID: service.ID,
		Name:      service.Name,
		From:      change.From,
		To:        change.To,
		Timestamp: time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("[WS] marshal_failed service=%s err=%v", service.Name, err)
		return
	}

	GlobalHub.Broadcast(payload)
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func (e *Engine) NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

		case msg := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					delete(h.clients, c)
					close(c.send)
				}
			}
		}
	}
}

func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}
