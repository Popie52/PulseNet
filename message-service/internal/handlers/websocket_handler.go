package handlers

import (
	"context"
	"message-service/pkg/db"
	"net/http"
	"sync"
	"time"
	"log"

	"github.com/gorilla/websocket"
)


type Client struct {
	conn *websocket.Conn
	send chan []byte 
	once sync.Once
}

type Hub struct {
	connections map[string][]*Client
	mu 			sync.RWMutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var hub = Hub{
	connections: make(map[string][]*Client),
}


func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("x-user-id")

	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return 
	}

	client := &Client {
		conn: conn,
		send: make(chan []byte, 50),
	}

	hub.mu.Lock()
	hub.connections[userID] = append(hub.connections[userID], client)
	hub.mu.Unlock()

	go writePump(userID, client)
	go readPump(userID, client)
}

func readPump(userID string, client *Client) {
	defer func() {
		removeConnection(userID, client)
		client.conn.Close()
	}()

	client.conn.SetReadLimit(512)
	client.conn.SetReadDeadline(time.Now().Add(60*time.Second))

	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60* time.Second))
		return nil
	})

	for {
		_, _, err := client.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}


func writePump(userID string, client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10*time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := client.conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				removeConnection(userID, client)
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				removeConnection(userID, client)
				return
			}	
		}
	}
}

func removeConnection(userID string, target *Client) {
	hub.Remove(userID, target)
	target.Close()
}


func broadcastToConversation(conversationID string, message []byte) {
	users, err := getParticipants(context.Background(), conversationID)

	if err != nil {
		log.Println("getParticipants error:", err)
		return
	}

	for _, userID := range users {
		hub.mu.RLock()
		clientsMap := hub.connections[userID]
		clients := append([]*Client(nil), clientsMap...)
		hub.mu.RUnlock()

		for _, client := range clients {
			select {
			case client.send <- message:
			default:
				removeConnection(userID, client)
			}
		}
	}
}


func getParticipants(ctx context.Context ,conversationID string) ([]string, error){
	rows, err := db.DB.QueryContext( 
		ctx,
		`SELECT user_id FROM conversation_participants WHERE conversation_id = $1`,
		conversationID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []string 

	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}

		users = append(users, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (c *Client) Close() {
	c.once.Do(
		func() {
			if c.send != nil {
				close(c.send)
				c.send = nil
			}

			if c.conn != nil {
				c.conn.Close()
				c.conn = nil
			}
		},
	)
}

func (h *Hub) Remove(userID string, target *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.connections[userID]

	for i, c := range clients {
		if c == target {
			h.connections[userID] = append(clients[:i], clients[i+1:]...)
			break
		} 
	}

	if len(h.connections[userID]) == 0 {
		delete(h.connections, userID)
	}

}