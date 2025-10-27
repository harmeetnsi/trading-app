package handlers

import (
	"fmt"
	"log" // FIX: Import the standard log package
	"net/http"
	"strconv"
	"strings"

	"trading-app/internal/database"
	"trading-app/internal/openalgo"
	"trading-app/pkg/utils"
)

type TradeHandler struct {
	db *database.DB
	// FIX 1: Add OpenAlgo Client
	openalgo *openalgo.OpenAlgoClient
}

// FIX 2: Update constructor to accept OpenAlgo Client
func NewTradeHandler(db *database.DB, openalgoClient *openalgo.OpenAlgoClient) *TradeHandler {
	return &TradeHandler{
		db: db,
		openalgo: openalgoClient,
	}
}

// GetTrades retrieves recent trades for the current user (Original Function)
func (h *TradeHandler) GetTrades(w http.ResponseWriter, r *http.Request) {
	// ... (Original logic for getting trades)
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

// FIX 3: New method to handle the /signal route for testing the Pine Script condition
func (h *TradeHandler) HandleSignal(w http.ResponseWriter, r *http.Request) {
	// 1. Extract required parameters from URL query
	symbol := r.URL.Query().Get("symbol")
	condition := r.URL.Query().Get("pine_condition")

	if symbol == "" || condition == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Missing 'symbol' or 'pine_condition' parameters")
		return
	}

	// --- NEW LOGIC: Read Exchange (Optional) ---
	// Get exchange from query, defaulting to "NSE" if not provided.
	exchange := r.URL.Query().Get("exchange")
	if exchange == "" {
		exchange = "NSE"
	}

	// 2. Call the core evaluation logic (Now includes the exchange!)
	// Note the addition of the 'exchange' argument here:
	isConditionMet, _, err := h.openalgo.EvaluatePineCondition(condition, strings.ToUpper(symbol), strings.ToUpper(exchange))
	if err != nil {
		// Log the error internally
		log.Printf("Signal evaluation failed for %s on %s: %v", symbol, exchange, err)
		// Return a clean error message to the user
		utils.ErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Signal evaluation failed: %v", err.Error()))
		return
	}

	// 3. Return the result
	result := map[string]interface{}{
		"symbol":     symbol,
		"exchange":   exchange, // Include exchange in the response
		"condition":  condition,
		"signal_met": isConditionMet,
		"message":    fmt.Sprintf("Condition '%s' for %s on %s is %t", condition, symbol, exchange, isConditionMet),
	}

	utils.SuccessResponse(w, "Signal evaluation complete", result)
}
