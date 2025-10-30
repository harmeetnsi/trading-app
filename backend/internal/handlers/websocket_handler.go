package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"trading-app/internal/ai"
	"trading-app/internal/auth"
	"trading-app/internal/database"
	"trading-app/internal/email"
	wsocket "trading-app/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

type WebSocketHandler struct {
	hub            *wsocket.Hub
	db             *database.DB
	aiClient       *ai.AIClient
	openalgoURL    string
	openalgoAPIKey string
	emailService   *email.EmailService
	emailRecipient string
}

func NewWebSocketHandler(hub *wsocket.Hub, db *database.DB, aiClient *ai.AIClient, openalgoURL string, openalgoAPIKey string, emailService *email.EmailService, emailRecipient string) *WebSocketHandler {
	return &WebSocketHandler{
		hub:            hub,
		db:             db,
		aiClient:       aiClient,
		openalgoURL:    openalgoURL,
		openalgoAPIKey: openalgoAPIKey,
		emailService:   emailService,
		emailRecipient: emailRecipient,
	}
}

// HandleWebSocket handles websocket connections
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "No token provided", http.StatusUnauthorized)
		return
	}

	// Validate token
	userID, err := auth.ValidateToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := wsocket.NewClient(
		h.hub,
		conn,
		userID,
		h.db,
		h.aiClient,
		h.openalgoURL,
		h.openalgoAPIKey,
		h.emailService,
		h.emailRecipient,
	)

	// Register client
	h.hub.Register <- client

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()
}
