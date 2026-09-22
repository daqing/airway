package websocket

import (
	"log"
	"net/http"

	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// sameOriginOnly flips the upgrader from the permissive web default to
// same-origin-only; desktop builds enable it via CheckSameOrigin.
var sameOriginOnly bool

// CheckSameOrigin restricts future upgrades to requests whose Origin host
// matches the request Host. Desktop windows load the app from its own
// 127.0.0.1 origin, so their connections still pass; cross-origin upgrades
// can only come from other local pages.
func CheckSameOrigin() {
	sameOriginOnly = true
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		if !sameOriginOnly {
			// Allow all connections for this example. In production, restrict origins.
			return true
		}
		return utils.SameOriginRequest(r)
	},
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

// Broadcast sends msg to every connected WebSocket client. Used by the
// frontend dev server to push livereload notifications.
func Broadcast(msg string) {
	hub.ch <- msg
}
