
package handlers

import (
	"net/http"
	"strconv"

	"trading-app/internal/database"
	"trading-app/pkg/utils"
)

type TradeHandler struct {
	db *database.DB
}

func NewTradeHandler(db *database.DB) *TradeHandler {
	return &TradeHandler{db: db}
}

// GetTrades retrieves recent trades for the current user
func (h *TradeHandler) GetTrades(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	limitStr := r.URL.Query().Get("limit")
	limit := 50 // Default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	trades, err := h.db.GetTradesByUserID(userID, limit)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve trades")
		return
	}

	utils.SuccessResponse(w, "Trades retrieved", trades)
}
