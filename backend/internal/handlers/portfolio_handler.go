
package handlers

import (
        "net/http"

        "trading-app/internal/database"
        "trading-app/internal/models"
        "trading-app/internal/openalgo"
        "trading-app/pkg/utils"
)

type PortfolioHandler struct {
        db          *database.DB
        openalgo    *openalgo.Client
}

func NewPortfolioHandler(db *database.DB, openalgoClient *openalgo.Client) *PortfolioHandler {
        return &PortfolioHandler{
                db:       db,
                openalgo: openalgoClient,
        }
}

// GetPortfolio retrieves portfolio data from OpenAlgo
func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
        portfolio, err := h.openalgo.CalculatePortfolio()
        if err != nil {
                utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve portfolio: "+err.Error())
                return
        }

        utils.SuccessResponse(w, "Portfolio retrieved", portfolio)
}

// GetPositions retrieves current positions
func (h *PortfolioHandler) GetPositions(w http.ResponseWriter, r *http.Request) {
        positions, err := h.openalgo.GetPositions()
        if err != nil {
                utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve positions: "+err.Error())
                return
        }

        utils.SuccessResponse(w, "Positions retrieved", positions)
}

// GetHoldings retrieves current holdings
func (h *PortfolioHandler) GetHoldings(w http.ResponseWriter, r *http.Request) {
        holdings, err := h.openalgo.GetHoldings()
        if err != nil {
                utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve holdings: "+err.Error())
                return
        }

        utils.SuccessResponse(w, "Holdings retrieved", holdings)
}

// PlaceOrder places a new order
func (h *PortfolioHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
        userID := r.Context().Value("user_id").(int)

        var order openalgo.Order
        if err := utils.ParseJSON(r, &order); err != nil {
                utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
                return
        }

        // Validate order
        if order.Symbol == "" || order.Exchange == "" || order.Action == "" || order.Quantity <= 0 {
                utils.ErrorResponse(w, http.StatusBadRequest, "Invalid order data")
                return
        }

        // Place order via OpenAlgo
        response, err := h.openalgo.PlaceOrder(order)
        if err != nil {
                utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to place order: "+err.Error())
                return
        }

        // Save trade to database
        trade := &models.Trade{
                UserID:    userID,
                Symbol:    order.Symbol,
                Action:    order.Action,
                Quantity:  order.Quantity,
                Price:     order.Price,
                OrderType: order.OrderType,
                Status:    "pending",
                OrderID:   response.OrderID,
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

        quote, err := h.openalgo.GetQuote(symbol, exchange)
        if err != nil {
                utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve quote: "+err.Error())
                return
        }

        utils.SuccessResponse(w, "Quote retrieved", quote)
}
