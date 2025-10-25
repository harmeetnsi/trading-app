
package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"trading-app/internal/ai"
	"trading-app/internal/database"
	"trading-app/internal/models"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512 KB
)

// Client represents a websocket client
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID int
	db     *database.DB
	ai     *ai.AIClient
}

// Message represents a websocket message
type Message struct {
	Type    string      `json:"type"` // "chat", "typing", "ping"
	Content string      `json:"content,omitempty"`
	FileID  *int        `json:"file_id,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// NewClient creates a new websocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID int, db *database.DB, aiClient *ai.AIClient) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		db:     db,
		ai:     aiClient,
	}
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	c.conn.SetReadLimit(maxMessageSize)

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse message
		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Handle different message types
		switch msg.Type {
		case "chat":
			c.handleChatMessage(&msg)
		case "typing":
			// Echo typing indicator to user (for multi-device support later)
			c.send <- messageBytes
		case "ping":
			// Send pong
			pong := Message{Type: "pong"}
			pongBytes, _ := json.Marshal(pong)
			c.send <- pongBytes
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
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

// handleChatMessage handles incoming chat messages
func (c *Client) handleChatMessage(msg *Message) {
	// Save user message to database
	chatMsg := &models.ChatMessage{
		UserID:  c.userID,
		Role:    "user",
		Content: msg.Content,
		FileID:  msg.FileID,
	}

	savedMsg, err := c.db.CreateChatMessage(chatMsg)
	if err != nil {
		log.Printf("Failed to save chat message: %v", err)
		c.sendError("Failed to save message")
		return
	}

	// Echo user message back
	userMsgResponse := Message{
		Type:    "chat",
		Content: savedMsg.Content,
		Data: map[string]interface{}{
			"id":         savedMsg.ID,
			"role":       "user",
			"created_at": savedMsg.CreatedAt,
		},
	}
	userMsgBytes, _ := json.Marshal(userMsgResponse)
	c.send <- userMsgBytes

	// Send typing indicator
	typingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": true}}
	typingBytes, _ := json.Marshal(typingMsg)
	c.send <- typingBytes

	// Get AI response
	go c.processAIResponse(msg.Content, msg.FileID)
}

// processAIResponse gets AI response and sends it to the client
func (c *Client) processAIResponse(userMessage string, fileID *int) {
	// Get chat history for context
	history, err := c.db.GetChatMessagesByUserID(c.userID, 10)
	if err != nil {
		log.Printf("Failed to get chat history: %v", err)
	}

	// Get file context if fileID is provided
	var fileContext string
	if fileID != nil {
		file, err := c.db.GetFileByID(*fileID)
		if err == nil && file != nil {
			fileContext = file.ProcessedData
		}
	}

	// Build context for AI
	context := c.ai.BuildContext(history, fileContext)

	// Get AI response
	aiResponse, err := c.ai.GetChatResponse(userMessage, context)
	if err != nil {
		log.Printf("Failed to get AI response: %v", err)
		aiResponse = "I apologize, but I'm having trouble processing your request right now. Please try again."
	}

	// Save AI response to database
	aiMsg := &models.ChatMessage{
		UserID:  c.userID,
		Role:    "assistant",
		Content: aiResponse,
	}

	savedAIMsg, err := c.db.CreateChatMessage(aiMsg)
	if err != nil {
		log.Printf("Failed to save AI message: %v", err)
	}

	// Stop typing indicator
	typingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": false}}
	typingBytes, _ := json.Marshal(typingMsg)
	c.send <- typingBytes

	// Send AI response
	aiMsgResponse := Message{
		Type:    "chat",
		Content: aiResponse,
		Data: map[string]interface{}{
			"id":         savedAIMsg.ID,
			"role":       "assistant",
			"created_at": savedAIMsg.CreatedAt,
		},
	}
	aiMsgBytes, _ := json.Marshal(aiMsgResponse)
	c.send <- aiMsgBytes
}

// sendError sends an error message to the client
func (c *Client) sendError(errMsg string) {
	errorMsg := Message{
		Type: "error",
		Data: map[string]string{"message": errMsg},
	}
	errorBytes, _ := json.Marshal(errorMsg)
	c.send <- errorBytes
}
