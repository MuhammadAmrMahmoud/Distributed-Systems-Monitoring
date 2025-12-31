package service

import (
	"Distributed-Health-Monitoring/models"
	"encoding/json"
	"log"
	"time"
)

var GlobalHub *Hub

func BroadcastStateChange(
	service models.ExternalService,
	change *models.StateChange,
) {
	event := models.ServiceStateChangeEvent{
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
	clients    map[*models.Client]bool
	broadcast  chan []byte
	register   chan *models.Client
	unregister chan *models.Client
}

func (e *Engine) NewHub() *Hub {
	return &Hub{
		clients:    make(map[*models.Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *models.Client),
		unregister: make(chan *models.Client),
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
				close(client.Send)
			}

		case msg := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.Send <- msg:
				default:
					delete(h.clients, c)
					close(c.Send)
				}
			}
		}
	}
}

func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}
