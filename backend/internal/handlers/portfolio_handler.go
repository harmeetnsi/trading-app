package handlers

import (
	//"fmt" 
	"log"
	"net/http"
	"strings"

	"trading-app/internal/database"
	"trading-app/internal/models"
	"trading-app/internal/openalgo"
	"trading-app/pkg/utils"
)

type PortfolioHandler struct {
	db       *database.DB
	openalgo *openalgo.OpenAlgoClient
}

func NewPortfolioHandler(db *database.DB, openalgoClient *openalgo.OpenAlgoClient) *PortfolioHandler {
	return &PortfolioHandler{
		db:       db,
		openalgo: openalgoClient,
	}
}

// GetPortfolio retrieves portfolio data from OpenAlgo (Kept as Not Implemented)
func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	utils.ErrorResponse(w, http.StatusNotImplemented, "GetPortfolio is not yet implemented after refactoring.")
}

// GetPositions retrieves current positions (Kept as Not Implemented)
func (h *PortfolioHandler) GetPositions(w http.ResponseWriter, r *http.Request) {
	utils.ErrorResponse(w, http.StatusNotImplemented, "GetPositions is not yet implemented after refactoring.")
}

// GetHoldings retrieves current holdings (Kept as Not Implemented)
func (h *PortfolioHandler) GetHoldings(w http.ResponseWriter, r *http.Request) {
	utils.ErrorResponse(w, http.StatusNotImplemented, "GetHoldings is not yet implemented after refactoring.")
}

// PlaceOrder places a new order
func (h *PortfolioHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var orderReq openalgo.OpenAlgoSmartOrderRequest
	if err := utils.ParseJSON(r, &orderReq); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate order
	if orderReq.Symbol == "" || orderReq.Exchange == "" || orderReq.Action == "" || orderReq.Quantity <= 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid order data")
		return
	}

	// Place order via OpenAlgo
	response, err := h.openalgo.PlaceOpenAlgoSmartOrder(&orderReq)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to place order: "+err.Error())
		return
	}

	// Save trade to database
	trade := &models.Trade{
		UserID:    userID,
		Symbol:    orderReq.Symbol,
		Action:    orderReq.Action,
		Quantity:  orderReq.Quantity,
		Price:     orderReq.Price,
		OrderType: orderReq.Pricetype,
		Status:    "pending",
		OrderID:   response.Data.OrderID,
	}

	savedTrade, err := h.db.CreateTrade(trade)
	if err != nil {
		utils.SuccessResponse(w, "Order placed", response)
		return
	}

	utils.SuccessResponse(w, "Order placed successfully", map[string]interface{}{
		"order": response,
		"trade": savedTrade,
	})
}

// GetQuote retrieves market quote
func (h *PortfolioHandler) GetQuote(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	exchange := r.URL.Query().Get("exchange")

	if symbol == "" || exchange == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Symbol and exchange are required")
		return
	}

	quote, err := h.openalgo.FetchOpenAlgoQuote(strings.ToUpper(symbol), strings.ToUpper(exchange)) 
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve quote: "+err.Error())
		return
	}

	utils.SuccessResponse(w, "Quote retrieved", quote)
}

// HandlePortfolioValue retrieves the current valuation of the user's portfolio
func (h *PortfolioHandler) HandlePortfolioValue(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	// Get exchange from query parameter, default to NSE
	exchange := r.URL.Query().Get("exchange")
	if exchange == "" {
		exchange = "NSE"
	}
	exchange = strings.ToUpper(exchange)

	positions, err := h.db.GetOpenPositionsByUserID(userID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve portfolio positions")
		return
	}

	var totalPortfolioValue float64
	for _, pos := range positions {
		quote, err := h.openalgo.FetchOpenAlgoQuote(pos.Symbol, exchange)
		if err != nil {
			log.Printf("Warning: Failed to fetch quote for %s on %s: %v", pos.Symbol, exchange, err)
			continue 
		}
		totalPortfolioValue += quote.LTP * float64(pos.Quantity)
	}

	result := map[string]interface{}{
		"total_value":    totalPortfolioValue,
		"position_count": len(positions),
		"exchange":       exchange,
	}

	utils.SuccessResponse(w, "Portfolio valuation complete", result)
}

// HandlePortfolioSignal checks if a condition is met for every symbol in the portfolio
func (h *PortfolioHandler) HandlePortfolioSignal(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	condition := r.URL.Query().Get("pine_condition")
	if condition == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Missing 'pine_condition' parameter")
		return
	}

	// Get exchange and interval from query parameters
	exchange := r.URL.Query().Get("exchange")
	if exchange == "" {
		exchange = "NSE"
	}
	exchange = strings.ToUpper(exchange)

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

	positions, err := h.db.GetOpenPositionsByUserID(userID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve portfolio positions")
		return
	}

	if len(positions) == 0 {
		utils.SuccessResponse(w, "No open positions found in portfolio", nil)
		return
	}

	signalResults := make(map[string]bool)

	for _, pos := range positions {
		symbol := pos.Symbol
		isMet, _, err := h.openalgo.EvaluatePineCondition(interval, condition, strings.ToUpper(symbol), exchange)
		if err != nil {
			log.Printf("Signal evaluation failed for %s on %s (%s): %v", symbol, exchange, interval, err)
			signalResults[symbol] = false 
			continue
		}
		signalResults[symbol] = isMet
	}

	result := map[string]interface{}{
		"condition": condition,
		"exchange":  exchange,
		"interval":  interval,
		"results":   signalResults,
	}

	utils.SuccessResponse(w, "Portfolio signal evaluation complete", result)
}

// HandleSignalTest is the unprotected test route for /signal
func (h *PortfolioHandler) HandleSignalTest(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	condition := r.URL.Query().Get("pine_condition")
	exchange := r.URL.Query().Get("exchange")
	interval := r.URL.Query().Get("interval")

	// Defaults
	if exchange == "" {
		exchange = "NSE"
	}
	exchange = strings.ToUpper(exchange)

	if interval == "" {
		interval = "5m"
	}
	interval = strings.ToLower(interval)

	if symbol == "" || condition == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Symbol and pine_condition are required")
		return
	}

	// Validate interval
	if interval != "5m" && interval != "15m" && interval != "1h" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid interval. Use 5m, 15m, or 1h")
		return
	}

	isConditionMet, indicatorValues, err := h.openalgo.EvaluatePineCondition(interval, condition, strings.ToUpper(symbol), exchange)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Evaluation failed: "+err.Error())
		return
	}

	result := map[string]interface{}{
		"condition_met":    isConditionMet,
		"symbol":           symbol,
		"condition":        condition,
		"exchange":         exchange,
		"interval":         interval,
		"indicator_values": indicatorValues,
	}

	utils.SuccessResponse(w, "Signal evaluation complete", result)
}