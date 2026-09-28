package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan Message
	userID    string
	role      string
	vehicleID string
	channels  map[string]bool // subscribed channels
}

type Hub struct {
	clients    map[*Client]bool
	byUserID   map[string]*Client
	byRole     map[string]map[*Client]bool
	byVehicle  map[string]map[*Client]bool
	byChannel  map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
	redis      *redis.Client
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
}

func NewHub(redisClient *redis.Client) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:    make(map[*Client]bool),
		byUserID:   make(map[string]*Client),
		byRole:     make(map[string]map[*Client]bool),
		byVehicle:  make(map[string]map[*Client]bool),
		byChannel:  make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message, 256),
		redis:      redisClient,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (h *Hub) Run() {
	// Subscribe to Redis channels for cross-instance messaging
	pubsub := h.redis.Subscribe(h.ctx, "aegisroad:broadcast", "aegisroad:role:*", "aegisroad:vehicle:*")
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true

			if client.userID != "" {
				h.byUserID[client.userID] = client
			}
			if client.role != "" {
				if h.byRole[client.role] == nil {
					h.byRole[client.role] = make(map[*Client]bool)
				}
				h.byRole[client.role][client] = true
			}
			if client.vehicleID != "" {
				if h.byVehicle[client.vehicleID] == nil {
					h.byVehicle[client.vehicleID] = make(map[*Client]bool)
				}
				h.byVehicle[client.vehicleID][client] = true
			}
			h.mu.Unlock()

			log.Info().Str("user_id", client.userID).Str("role", client.role).Msg("Client connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				if client.userID != "" {
					delete(h.byUserID, client.userID)
				}
				if client.role != "" {
					delete(h.byRole[client.role], client)
				}
				if client.vehicleID != "" {
					delete(h.byVehicle[client.vehicleID], client)
				}
				close(client.send)
			}
			h.mu.Unlock()

			log.Info().Str("user_id", client.userID).Msg("Client disconnected")

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()

		case redisMsg := <-ch:
			var msg Message
			if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err == nil {
				h.mu.RLock()
				for client := range h.clients {
					select {
					case client.send <- msg:
					default:
					}
				}
				h.mu.RUnlock()
			}
		}
	}
}

func (h *Hub) HandleWebSocket(c *gin.Context) {
	userID := c.GetString("user_id")
	role := c.GetString("user_role")
	vehicleID := c.Query("vehicle_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}

	client := &Client{
		hub:       h,
		conn:      conn,
		send:      make(chan Message, 256),
		userID:    userID,
		role:      role,
		vehicleID: vehicleID,
		channels:  make(map[string]bool),
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (h *Hub) BroadcastAll(msg Message) {
	h.broadcast <- msg

	// Also publish to Redis for other instances
	data, _ := json.Marshal(msg)
	h.redis.Publish(h.ctx, "aegisroad:broadcast", data)
}

func (h *Hub) BroadcastToRole(role string, msg Message) {
	h.mu.RLock()
	clients := h.byRole[role]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- msg:
		default:
		}
	}

	// Publish to Redis for other instances
	data, _ := json.Marshal(msg)
	h.redis.Publish(h.ctx, "aegisroad:role:"+role, data)
}

func (h *Hub) BroadcastToVehicle(vehicleID string, msg Message) {
	h.mu.RLock()
	clients := h.byVehicle[vehicleID]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- msg:
		default:
		}
	}

	data, _ := json.Marshal(msg)
	h.redis.Publish(h.ctx, "aegisroad:vehicle:"+vehicleID, data)
}

func (h *Hub) BroadcastToUser(userID string, msg Message) {
	h.mu.RLock()
	client, ok := h.byUserID[userID]
	h.mu.RUnlock()

	if ok {
		select {
		case client.send <- msg:
		default:
		}
	}
}

func (h *Hub) SubscribeChannel(client *Client, channel string) {
	h.mu.Lock()
	if h.byChannel[channel] == nil {
		h.byChannel[channel] = make(map[*Client]bool)
	}
	h.byChannel[channel][client] = true
	client.channels[channel] = true
	h.mu.Unlock()
}

func (h *Hub) UnsubscribeChannel(client *Client, channel string) {
	h.mu.Lock()
	delete(h.byChannel[channel], client)
	delete(client.channels, channel)
	h.mu.Unlock()
}

func (h *Hub) BroadcastToChannel(channel string, msg Message) {
	h.mu.RLock()
	clients := h.byChannel[channel]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- msg:
		default:
		}
	}

	data, _ := json.Marshal(msg)
	h.redis.Publish(h.ctx, "aegisroad:channel:"+channel, data)
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Msg("WebSocket read error")
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Error().Err(err).Msg("Invalid WebSocket message")
			continue
		}

		// Handle client messages (subscriptions, pings, etc.)
		switch msg.Type {
		case "subscribe":
			if channel, ok := msg.Data.(string); ok {
				c.hub.SubscribeChannel(c, channel)
			}
		case "unsubscribe":
			if channel, ok := msg.Data.(string); ok {
				c.hub.UnsubscribeChannel(c, channel)
			}
		case "ping":
			c.send <- Message{Type: "pong"}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Error().Err(err).Msg("Failed to marshal WebSocket message")
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Error().Err(err).Msg("WebSocket write error")
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) Close() {
	h.cancel()
	h.mu.Lock()
	for client := range h.clients {
		close(client.send)
	}
	h.mu.Unlock()
}