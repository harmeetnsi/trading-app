
package handlers

import (
	"net/http"
	"strconv"

	"trading-app/internal/database"
	"trading-app/internal/models"
	"trading-app/pkg/utils"
)

type ChatHandler struct {
	db *database.DB
}

func NewChatHandler(db *database.DB) *ChatHandler {
	return &ChatHandler{db: db}
}

type SendMessageRequest struct {
	Content string `json:"content"`
	FileID  *int   `json:"file_id,omitempty"`
}

// GetMessages retrieves chat history for the current user
func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	limitStr := r.URL.Query().Get("limit")
	limit := 50 // Default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := h.db.GetChatMessagesByUserID(userID, limit)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve messages")
		return
	}

	utils.SuccessResponse(w, "Messages retrieved", messages)
}

// SendMessage sends a new message (user message, AI response handled via WebSocket)
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req SendMessageRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Content == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Content is required")
		return
	}

	// Create user message
	msg := &models.ChatMessage{
		UserID:  userID,
		Role:    "user",
		Content: req.Content,
		FileID:  req.FileID,
	}

	message, err := h.db.CreateChatMessage(msg)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to save message")
		return
	}

	utils.SuccessResponse(w, "Message sent", message)
}
