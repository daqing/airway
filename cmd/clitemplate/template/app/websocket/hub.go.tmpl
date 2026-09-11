package websocket

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

var hub *Hub

type WebSocketClient struct {
	Id   int
	Conn *websocket.Conn
}

type Hub struct {
	id      int
	clients map[int]*WebSocketClient

	lock sync.Mutex
	ch   chan string
}

func init() {
	hub = &Hub{
		id:      0,
		clients: make(map[int]*WebSocketClient),
		ch:      make(chan string),
	}

	go hub.Run()
}

func (h *Hub) Run() {
	for msg := range h.ch {
		h.lock.Lock()
		for _, client := range h.clients {
			err := client.Conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				log.Println("Error writing message to client:", err)
				client.Conn.Close()
				delete(h.clients, client.Id)
			}
		}
		h.lock.Unlock()
	}
}

func (h *Hub) AddClient(conn *websocket.Conn) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.id++
	client := &WebSocketClient{
		Id:   h.id,
		Conn: conn,
	}

	h.clients[client.Id] = client
}

func (h *Hub) HandleMessages(conn *websocket.Conn) {
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		h.ch <- string(msg)
	}
}
