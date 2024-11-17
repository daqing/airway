package websocket

import (
	"log"
	"net/http"

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

func Conn(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade to WebSocket:", err)
		return
	}

	defer conn.Close()

	log.Println("WebSocket connection established")

	for {
		// Read a message
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		log.Printf("Received message: %s\n", msg)

		// Echo the message back to the client
		err = conn.WriteMessage(messageType, []byte("Echo: "+string(msg)))
		if err != nil {
			log.Println("Error writing message:", err)
			break
		}
	}
}
