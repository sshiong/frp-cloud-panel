package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该限制
	},
}

// Hub WebSocket 中心
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// Client WebSocket 客户端
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   uint
	clientID string
}

// Message WebSocket 消息
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// NewHub 创建新的 Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run 运行 Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected: %s (User: %d)", client.clientID, client.userID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected: %s (User: %d)", client.clientID, client.userID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToUser 向特定用户广播消息
func (h *Hub) BroadcastToUser(userID uint, message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// BroadcastToAll 向所有客户端广播消息
func (h *Hub) BroadcastToAll(message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	h.broadcast <- data
}

// NotifyConfigChange 通知配置变更
func (h *Hub) NotifyConfigChange(userID uint, clientID string, version int) {
	message := &Message{
		Type: "config_change",
		Payload: map[string]interface{}{
			"client_id": clientID,
			"version":   version,
			"timestamp": time.Now().Unix(),
		},
	}
	h.BroadcastToUser(userID, message)
}

// NotifyMappingStatus 通知映射状态变更
func (h *Hub) NotifyMappingStatus(userID uint, mappingID uint, status string) {
	message := &Message{
		Type: "mapping_status",
		Payload: map[string]interface{}{
			"mapping_id": mappingID,
			"status":     status,
			"timestamp":  time.Now().Unix(),
		},
	}
	h.BroadcastToUser(userID, message)
}

// NotifyClientStatus 通知客户端状态变更
func (h *Hub) NotifyClientStatus(userID uint, clientID string, status string) {
	message := &Message{
		Type: "client_status",
		Payload: map[string]interface{}{
			"client_id": clientID,
			"status":    status,
			"timestamp": time.Now().Unix(),
		},
	}
	h.BroadcastToUser(userID, message)
}

// HandleWebSocket 处理 WebSocket 连接
func (h *Hub) HandleWebSocket(c *gin.Context) {
	// 从查询参数获取认证信息
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
		return
	}

	// TODO: 验证 Token 并获取用户 ID
	// 这里简化处理，实际应该验证 JWT Token
	userID := uint(1) // 临时硬编码

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		clientID: c.Query("client_id"),
	}

	h.register <- client

	// 启动读写协程
	go client.writePump()
	go client.readPump()
}

// readPump 读取消息
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// 处理接收到的消息
		c.handleMessage(message)
	}
}

// writePump 写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 处理接收到的消息
func (c *Client) handleMessage(message []byte) {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return
	}

	switch msg.Type {
	case "ping":
		// 响应 pong
		pong := &Message{
			Type: "pong",
			Payload: map[string]interface{}{
				"timestamp": time.Now().Unix(),
			},
		}
		data, _ := json.Marshal(pong)
		c.send <- data

	case "subscribe":
		// 订阅频道
		log.Printf("Client %s subscribed to channel", c.clientID)

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// GetConnectedClients 获取连接的客户端数量
func (h *Hub) GetConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetConnectedClientsByUser 获取特定用户的连接客户端数量
func (h *Hub) GetConnectedClientsByUser(userID uint) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for client := range h.clients {
		if client.userID == userID {
			count++
		}
	}
	return count
}
