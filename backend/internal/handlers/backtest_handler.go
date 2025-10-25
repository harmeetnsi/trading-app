
package handlers

import (
	"net/http"
	"time"

	"trading-app/internal/database"
	"trading-app/internal/openalgo"
	"trading-app/internal/strategy"
	"trading-app/pkg/utils"
)

type BacktestHandler struct {
	db         *database.DB
	backtester *strategy.Backtester
}

func NewBacktestHandler(db *database.DB, openalgoClient *openalgo.Client) *BacktestHandler {
	return &BacktestHandler{
		db:         db,
		backtester: strategy.NewBacktester(db, openalgoClient),
	}
}

type RunBacktestRequest struct {
	StrategyID     int     `json:"strategy_id"`
	StartDate      string  `json:"start_date"`
	EndDate        string  `json:"end_date"`
	InitialCapital float64 `json:"initial_capital"`
	Symbol         string  `json:"symbol"`
	Exchange       string  `json:"exchange"`
}

// RunBacktest runs a backtest for a strategy
func (h *BacktestHandler) RunBacktest(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req RunBacktestRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate
	if req.StrategyID == 0 || req.InitialCapital <= 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid parameters")
		return
	}

	// Verify strategy ownership
	strat, err := h.db.GetStrategyByID(req.StrategyID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve strategy")
		return
	}
	if strat == nil {
		utils.ErrorResponse(w, http.StatusNotFound, "Strategy not found")
		return
	}
	if strat.UserID != userID {
		utils.ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid start date format (use YYYY-MM-DD)")
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid end date format (use YYYY-MM-DD)")
		return
	}

	// Run backtest
	params := strategy.BacktestParams{
		StrategyID:     req.StrategyID,
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: req.InitialCapital,
		Symbol:         req.Symbol,
		Exchange:       req.Exchange,
	}

	result, err := h.backtester.RunBacktest(params)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to run backtest: "+err.Error())
		return
	}

	utils.SuccessResponse(w, "Backtest completed", result)
}
