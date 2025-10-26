package websocket

import (
	"encoding/json"
	"fmt" // Keep fmt for Sprintf
	"log"
	// "net/http" // No longer needed here
	// "os" // No longer needed here
	"strconv"
	"strings"
	"time"
	// Keep math/indicator imports commented out for now
	// "math"
	// indicator "github.com/some-indicator-library" // Placeholder

	"github.com/gorilla/websocket"
	"trading-app/internal/ai"
	"trading-app/internal/database"
	"trading-app/internal/models"
	// Keep govaluate commented out
	// "github.com/Knetic/govaluate"
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

// --- REMOVED OpenAlgo Config (moved to openalgo_client.go) ---

// Client represents a websocket client
type Client struct {
	hub *Hub
	conn *websocket.Conn
	send chan []byte
	userID int
	db *database.DB
	ai *ai.AIClient
	// --- NEW: Add OpenAlgo Client ---
	oaClient *OpenAlgoClient
}

// Message represents a websocket message
type Message struct {
	Type    string      `json:"type"` // "chat", "typing", "ping", "error"
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
		// --- NEW: Initialize OpenAlgo Client ---
		oaClient: NewOpenAlgoClient(), // Assumes NewOpenAlgoClient is defined in openalgo_client.go
	}
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister <- c
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
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				// Error writing message (connection likely closed)
				return
			}

		case <-ticker.C:
			// Send ping message
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// Error sending ping (connection likely closed)
				return
			}
		}
	}
}

// handleChatMessage saves user message and decides whether to send to AI or handle as command
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

	// Echo user message back to the sender immediately
	userMsgResponse := Message{
		Type:    "chat",
		Content: savedMsg.Content,
		Data: map[string]interface{}{
			"id":         savedMsg.ID,
			"role":       "user",
			"created_at": savedMsg.CreatedAt,
			"file_id":    savedMsg.FileID,
		},
	}
	userMsgBytes, _ := json.Marshal(userMsgResponse)
	c.send <- userMsgBytes // Send only to the client 'c'

	// --- Check for command FIRST ---
	if strings.HasPrefix(msg.Content, "/") {
		// Handle trading command (runs in a goroutine)
		go c.handleTradingCommand(msg.Content)
	} else {
		// Not a command, send to AI for processing (runs in a goroutine)
		// Send typing indicator before calling AI
		typingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": true}}
		typingBytes, _ := json.Marshal(typingMsg)
		c.send <- typingBytes
		go c.processAIResponse(msg.Content, msg.FileID)
	}
}

// --- handleTradingCommand processes commands starting with "/" ---
func (c *Client) handleTradingCommand(command string) {
	// Send typing indicator
	typingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": true}}
	typingBytes, _ := json.Marshal(typingMsg)
	c.send <- typingBytes

	var responseContent string // This will hold the text response to send back

	parts := strings.Fields(command) // splits by space
	if len(parts) < 1 {
		responseContent = "Sorry, I didn't understand that command."
	} else {
		cmd := parts[0]
		switch cmd {
		case "/price":
			if len(parts) == 2 {
				symbol := strings.ToUpper(parts[1])
				// --- MODIFIED: Use oaClient ---
				quote, err := c.oaClient.fetchOpenAlgoQuote(symbol) // Calls function from openalgo_client.go
				if err != nil {
					log.Printf("Failed to fetch quote for %s: %v", symbol, err)
					responseContent = fmt.Sprintf("Sorry, I had trouble fetching the quote for %s: %v", symbol, err)
				} else {
					responseContent = fmt.Sprintf(
						"Here is the latest price for **%s**:\n\n---\n\n### **NSE: %s**\n| Metric | Value |\n| :--- | :--- |\n| **Last Traded Price** | **%.2f** |\n| **Change** | **%.2f (%.2f%%)** |\n| **Day's High** | %.2f |\n| **Day's Low** | %.2f |\n| **Open** | %.2f |\n| **Previous Close** | %.2f |\n\n*Data Source: Live feed from the National Stock Exchange (NSE) via OpenAlgo.*",
						symbol, symbol, quote.LTP, quote.Change, quote.ChangePercent, quote.High, quote.Low, quote.Open, quote.PreviousClose,
					)
				}
			} else {
				responseContent = "Usage: `/price <SYMBOL>` (e.g., `/price RELIANCE`)"
			}

		case "/buy_smart", "/sell_smart":
			if len(parts) == 3 {
				action := "BUY"
				if cmd == "/sell_smart" {
					action = "SELL"
				}
				symbol := strings.ToUpper(parts[1])
				quantityStr := parts[2]
				quantity, err := strconv.Atoi(quantityStr)
				if err != nil || quantity <= 0 {
					responseContent = "Invalid quantity. Must be a positive number."
				} else {
					orderReq := &OpenAlgoSmartOrderRequest{ // Struct definition is now in openalgo_client.go
						// Apikey: Handled by oaClient
						Strategy:     "manual_chat",
						Symbol:       symbol,
						Exchange:     "NSE",
						Action:       action,
						Pricetype:    "MARKET",
						Product:      "MIS",
						Quantity:     quantity,
						PositionSize: 0,
					}
					// --- MODIFIED: Use oaClient ---
					orderResponse, err := c.oaClient.placeOpenAlgoSmartOrder(orderReq) // Calls function from openalgo_client.go
					if err != nil {
						log.Printf("Failed to place %s smart order for %s: %v", action, symbol, err)
						responseContent = fmt.Sprintf("❌ Sorry, I had trouble placing your %s smart order: %v", action, err)
					} else {
						responseContent = fmt.Sprintf("✅ **Smart Order Submitted!**\n\n- **Action**: %s\n- **Symbol**: %s\n- **Qty**: %d\n- **Status**: %s\n- **Order ID**: %s",
							action, symbol, quantity, orderResponse.Status, orderResponse.Data.OrderID)
					}
				}
			} else if cmd == "/buy_smart" {
				responseContent = "Usage: `/buy_smart <SYMBOL> <QTY>`"
			} else {
				responseContent = "Usage: `/sell_smart <SYMBOL> <QTY>`"
			}

		case "/rsi":
			if len(parts) == 2 {
				symbol := strings.ToUpper(parts[1])
				responseContent = fmt.Sprintf("RSI calculation for %s is not implemented yet. Try `/price %s`.", symbol, symbol)
			} else {
				responseContent = "Usage: `/rsi <SYMBOL>`"
			}

		case "/signal":
			if len(parts) > 2 {
				symbol := strings.ToUpper(parts[1])
				condition := strings.Join(parts[2:], " ")
				log.Printf("Received signal command for %s with condition: %s", symbol, condition)
				// --- MODIFIED: Use oaClient ---
				isConditionMet, err := c.oaClient.evaluatePineCondition(condition, symbol) // Calls function from openalgo_client.go
				if err != nil {
					log.Printf("Error evaluating condition for %s: %v", symbol, err)
					responseContent = fmt.Sprintf("⚠️ Error evaluating signal for %s: %v", symbol, err)
				} else {
					if isConditionMet {
						responseContent = fmt.Sprintf("✅ Signal condition met for %s: `%s` (Evaluation Logic Placeholder)", symbol, condition)
					} else {
						responseContent = fmt.Sprintf("❌ Signal condition NOT met for %s: `%s` (Evaluation Logic Placeholder)", symbol, condition)
					}
				}
			} else {
				responseContent = "Usage: `/signal <SYMBOL> <PINE_SCRIPT_CONDITION>`"
			}

		default:
			responseContent = "Sorry, I didn't understand that command. Try `/price <SYMBOL>`, `/buy_smart <SYMBOL> <QTY>`, or `/signal <SYMBOL> <CONDITION>`."
		}
	}

	// --- Save and Send Response (common for all commands) ---
	assistMsg := &models.ChatMessage{
		UserID:  c.userID,
		Role:    "assistant",
		Content: responseContent,
	}
	var savedAssistMsg *models.ChatMessage
	savedAssistMsg, err := c.db.CreateChatMessage(assistMsg)
	if err != nil {
		log.Printf("Failed to save command response: %v", err)
		savedAssistMsg = &models.ChatMessage{ID: 0, CreatedAt: time.Now()}
	}

	// Stop typing indicator
	stopTypingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": false}}
	stopTypingBytes, _ := json.Marshal(stopTypingMsg)
	c.send <- stopTypingBytes

	// Send final response
	assistMsgResponse := Message{
		Type:    "chat",
		Content: responseContent,
		Data: map[string]interface{}{
			"id":         savedAssistMsg.ID,
			"role":       "assistant",
			"created_at": savedAssistMsg.CreatedAt,
		},
	}
	assistMsgBytes, _ := json.Marshal(assistMsgResponse)
	c.send <- assistMsgBytes
}

// --- REMOVED OpenAlgo Structs and API call functions (moved to openalgo_client.go) ---


// processAIResponse gets AI response and sends it to the client
func (c *Client) processAIResponse(userMessage string, fileID *int) {
	// Get chat history for context
	history, err := c.db.GetChatMessagesByUserID(c.userID, 10) // Get last 10 messages
	if err != nil {
		log.Printf("Failed to get chat history: %v", err)
		// Don't return, proceed without history
	}

	// Get file context if fileID is provided
	var fileContext string
	if fileID != nil {
		file, err := c.db.GetFileByID(*fileID)
		if err == nil && file != nil {
			// TODO: Potentially truncate file context if too long for AI
			fileContext = file.ProcessedData
		} else if err != nil {
			log.Printf("Failed to get file context for file ID %d: %v", *fileID, err)
		}
	}

	// Build context string for AI
	context := c.ai.BuildContext(history, fileContext)

	// Get AI response
	aiResponse, err := c.ai.GetChatResponse(userMessage, context)
	if err != nil {
		log.Printf("Failed to get AI response: %v", err)
		// Provide a user-friendly error message
		aiResponse = "I apologize, but I encountered an issue while processing your request with the AI. Please try again."
		// Don't save this generic error message to DB usually, unless needed
	} else {
		// Save *successful* AI response to database
		aiMsg := &models.ChatMessage{
			UserID:  c.userID,
			Role:    "assistant",
			Content: aiResponse,
		}
		savedAIMsg, saveErr := c.db.CreateChatMessage(aiMsg) // Capture saved message
		if saveErr != nil {
			log.Printf("Failed to save AI message: %v", saveErr)
			// Don't stop the user from getting the response
		} else {
		  // --- Use the actual ID from the saved message ---
			aiMsgResponse := Message{
				Type:    "chat",
				Content: aiResponse,
				Data: map[string]interface{}{
					"id":         savedAIMsg.ID, // Use actual saved ID
					"role":       "assistant",
					"created_at": savedAIMsg.CreatedAt, // Use timestamp from saved message
				},
			}
			aiMsgBytes, _ := json.Marshal(aiMsgResponse)
			// Stop typing indicator *before* sending the response
			stopTypingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": false}}
			stopTypingBytes, _ := json.Marshal(stopTypingMsg)
			c.send <- stopTypingBytes
			c.send <- aiMsgBytes // Send the AI response
			return // Exit after successfully sending AI response
		}
	}

	// Stop typing indicator (also runs if AI fails or saving fails)
	stopTypingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": false}}
	stopTypingBytes, _ := json.Marshal(stopTypingMsg)
	c.send <- stopTypingBytes

	// Send AI response back to the chat client (even if saving failed)
	aiMsgResponse := Message{
		Type:    "chat",
		Content: aiResponse, // Send the AI response (or error message)
		Data: map[string]interface{}{
			"id":         0,          // Use 0 if saving failed
			"role":       "assistant",
			"created_at": time.Now(),
		},
	}
	aiMsgBytes, _ := json.Marshal(aiMsgResponse)
	c.send <- aiMsgBytes
}


// sendError sends a structured error message to the client
func (c *Client) sendError(errMsg string) {
	errorMsg := Message{
		Type: "error", // Use a specific type for errors
		Data: map[string]string{"message": errMsg},
	}
	errorBytes, err := json.Marshal(errorMsg)
	if err == nil {
		c.send <- errorBytes
	} else {
		log.Printf("Failed to marshal error message: %v", err)
	}
}

