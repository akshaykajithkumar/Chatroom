package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type ChatRoom struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.Mutex
}

func NewChatRoom() *ChatRoom {
	return &ChatRoom{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (c *ChatRoom) run() {
	for {
		select {
		case client := <-c.register:
			c.mutex.Lock()
			c.clients[client] = true
			c.mutex.Unlock()
		case client := <-c.unregister:
			c.mutex.Lock()
			delete(c.clients, client)
			c.mutex.Unlock()
		case message := <-c.broadcast:
			c.mutex.Lock()
			for client := range c.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					fmt.Println("Error broadcasting message:", err)
					client.Close()
					delete(c.clients, client)
				}
			}
			c.mutex.Unlock()
		}
	}
}

func handleChatRoom(chatRoom *ChatRoom, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	chatRoom.register <- conn
	defer func() {
		chatRoom.unregister <- conn
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}
		chatRoom.broadcast <- message
	}
}

func main() {
	chatRoom := NewChatRoom()
	go chatRoom.run()

	r := gin.Default()

	r.LoadHTMLGlob("templates/*") // Load HTML templates from the "templates" folder

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "main.html", nil) // Render the "chat.html" template
	})

	r.GET("/ws", func(c *gin.Context) {
		handleChatRoom(chatRoom, c)
	})

	fmt.Println("Chat server is running on :8080")
	err := r.Run(":8080")
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
