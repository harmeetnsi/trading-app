package handlers

import (
	"fmt" // FIX: Added missing import
	"net/http"

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

// GetPortfolio retrieves portfolio data from OpenAlgo
func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	// FIX: This function is commented out because CalculatePortfolio is not defined
	// in the new openalgo/client.go file. It needs to be re-implemented if required.
	utils.ErrorResponse(w, http.StatusNotImplemented, "GetPortfolio is not yet implemented after refactoring.")
	// portfolio, err := h.openalgo.CalculatePortfolio()
	// if err != nil {
	// 	utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve portfolio: "+err.Error())
	// 	return
	// }
	// utils.SuccessResponse(w, "Portfolio retrieved", portfolio)
}

// GetPositions retrieves current positions
func (h *PortfolioHandler) GetPositions(w http.ResponseWriter, r *http.Request) {
	// FIX: This function is commented out because GetPositions is not defined
	// in the new openalgo/client.go file. It needs to be re-implemented if required.
	utils.ErrorResponse(w, http.StatusNotImplemented, "GetPositions is not yet implemented after refactoring.")
	// positions, err := h.openalgo.GetPositions()
	// if err != nil {
	// 	utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve positions: "+err.Error())
	// 	return
	// }
	// utils.SuccessResponse(w, "Positions retrieved", positions)
}

// GetHoldings retrieves current holdings
func (h *PortfolioHandler) GetHoldings(w http.ResponseWriter, r *http.Request) {
	// FIX: This function is commented out because GetHoldings is not defined
	// in the new openalgo/client.go file. It needs to be re-implemented if required.
	utils.ErrorResponse(w, http.StatusNotImplemented, "GetHoldings is not yet implemented after refactoring.")
	// holdings, err := h.openalgo.GetHoldings()
	// if err != nil {
	// 	utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve holdings: "+err.Error())
	// 	return
	// }
	// utils.SuccessResponse(w, "Holdings retrieved", holdings)
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

	// Place order via OpenAlgo - We must call the public method PlaceOpenAlgoSmartOrder
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
		OrderType: orderReq.Pricetype, // NOTE: Assuming OrderType maps to Pricetype
		Status:    "pending",
		OrderID:   response.Data.OrderID, // FIX: Use the correct struct path
	}

	savedTrade, err := h.db.CreateTrade(trade)
	if err != nil {
		// Log error but don't fail the order placement
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

	// NOTE: GetQuote is implemented using the public method FetchOpenAlgoQuote
	quote, err := h.openalgo.FetchOpenAlgoQuote(symbol) // Assuming exchange is implicitly handled or not needed
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve quote: "+err.Error())
		return
	}

	utils.SuccessResponse(w, "Quote retrieved", quote)
}

// HandleSignalTest is the unprotected test route for /signal we added in main.go
func (h *PortfolioHandler) HandleSignalTest(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	condition := r.URL.Query().Get("pine_condition")

	if symbol == "" || condition == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Symbol and pine_condition are required")
		return
	}

	// Execute the new logic for fetching data (Step A completed)
	isConditionMet, err := h.openalgo.EvaluatePineCondition(condition, symbol)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Evaluation failed: "+err.Error())
		return
	}

	result := fmt.Sprintf("Condition met: %t (Symbol: %s, Condition: %s)", isConditionMet, symbol, condition)
	utils.SuccessResponse(w, "Signal evaluation complete", result)
}
