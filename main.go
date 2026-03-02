package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleConnections(hub *Hub, w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	hub.register <- ws

	defer func() {
		hub.unregister <- ws
	}()

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}
		hub.broadcast <- message
	}
}

type Hub struct {
	clients    map[*websocket.Conn]Judge
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan []byte
	totalScore int
}

type Judge struct {
	ID      string
	Name    string
	InLobby bool
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = Judge{Name: "Anonymous"}
			fmt.Println("New Judge has entered the lobby!")

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
				fmt.Println("Judge has left the lobby")
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				client.WriteMessage(websocket.TextMessage, message)
			}

		}
	}
}

func main() {

	hub := &Hub{
		clients:    make(map[*websocket.Conn]Judge),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan []byte),
	}

	go hub.run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleConnections(hub, w, r)
	})

	fmt.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
