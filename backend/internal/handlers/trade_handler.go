package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"trading-app/internal/database"
	"trading-app/internal/openalgo"
	"trading-app/pkg/utils"
)

type TradeHandler struct {
	db       *database.DB
	openalgo *openalgo.OpenAlgoClient
}

func NewTradeHandler(db *database.DB, openalgoClient *openalgo.OpenAlgoClient) *TradeHandler {
	return &TradeHandler{
		db:       db,
		openalgo: openalgoClient,
	}
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

// HandleSignal handles the /signal route for testing Pine Script conditions
func (h *TradeHandler) HandleSignal(w http.ResponseWriter, r *http.Request) {
	// Extract required parameters from URL query
	symbol := r.URL.Query().Get("symbol")
	condition := r.URL.Query().Get("pine_condition")

	if symbol == "" || condition == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Missing 'symbol' or 'pine_condition' parameters")
		return
	}

	// Get exchange from query, default to NSE
	exchange := r.URL.Query().Get("exchange")
	if exchange == "" {
		exchange = "NSE"
	}
	exchange = strings.ToUpper(exchange)

	// Get interval from query, default to 5m
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "5m"
	}
	interval = strings.ToLower(interval)

	// Validate interval
	if interval != "5m" && interval != "15m" && interval != "1h" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid interval. Use 5m, 15m, or 1h")
		return
	}

	// Call the evaluation logic with interval
	isConditionMet, indicatorValues, err := h.openalgo.EvaluatePineCondition(interval, condition, strings.ToUpper(symbol), exchange)
	if err != nil {
		log.Printf("Signal evaluation failed for %s on %s (%s): %v", symbol, exchange, interval, err)
		utils.ErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Signal evaluation failed: %v", err.Error()))
		return
	}

	// Return the result with indicator values
	result := map[string]interface{}{
		"symbol":           symbol,
		"exchange":         exchange,
		"interval":         interval,
		"condition":        condition,
		"signal_met":       isConditionMet,
		"indicator_values": indicatorValues,
		"message":          fmt.Sprintf("Condition '%s' for %s on %s (%s) is %t", condition, symbol, exchange, interval, isConditionMet),
	}

	utils.SuccessResponse(w, "Signal evaluation complete", result)
}