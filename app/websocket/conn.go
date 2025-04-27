package websocket

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections for this example. In production, restrict origins.
		return true
	},
}

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

var hub *Hub

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
	h.id++
	h.lock.Lock()
	defer h.lock.Unlock()
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

func Conn(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade to WebSocket:", err)
		return
	}

	log.Println("WebSocket connection established")
	hub.AddClient(conn)

	go hub.HandleMessages(conn)
}

func Publish(c *gin.Context) {
	message := c.PostForm("message")
	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message cannot be empty"})
		return
	}

	hub.ch <- message
	c.JSON(http.StatusOK, gin.H{"status": "Message sent"})
}
