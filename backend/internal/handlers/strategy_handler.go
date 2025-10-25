
package handlers

import (
	"net/http"
	"strconv"

	"trading-app/internal/database"
	"trading-app/internal/models"
	"trading-app/pkg/utils"
)

type StrategyHandler struct {
	db *database.DB
}

func NewStrategyHandler(db *database.DB) *StrategyHandler {
	return &StrategyHandler{db: db}
}

type CreateStrategyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	FileID      int    `json:"file_id"`
	Code        string `json:"code"`
}

type UpdateStrategyStatusRequest struct {
	Status string `json:"status"` // "active", "paused", "stopped"
}

// GetStrategies retrieves all strategies for the current user
func (h *StrategyHandler) GetStrategies(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	strategies, err := h.db.GetStrategiesByUserID(userID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve strategies")
		return
	}

	utils.SuccessResponse(w, "Strategies retrieved", strategies)
}

// GetStrategy retrieves a specific strategy
func (h *StrategyHandler) GetStrategy(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Strategy ID is required")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	strategy, err := h.db.GetStrategyByID(id)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve strategy")
		return
	}
	if strategy == nil {
		utils.ErrorResponse(w, http.StatusNotFound, "Strategy not found")
		return
	}

	// Ensure user owns this strategy
	if strategy.UserID != userID {
		utils.ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	utils.SuccessResponse(w, "Strategy retrieved", strategy)
}

// CreateStrategy creates a new trading strategy
func (h *StrategyHandler) CreateStrategy(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req CreateStrategyRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.Code == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Name and code are required")
		return
	}

	strategy := &models.Strategy{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		FileID:      req.FileID,
		Code:        req.Code,
		Status:      "paused",
	}

	created, err := h.db.CreateStrategy(strategy)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create strategy")
		return
	}

	utils.SuccessResponse(w, "Strategy created successfully", created)
}

// UpdateStrategyStatus updates a strategy's status
func (h *StrategyHandler) UpdateStrategyStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Strategy ID is required")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	var req UpdateStrategyStatusRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Status != "active" && req.Status != "paused" && req.Status != "stopped" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid status")
		return
	}

	// Verify ownership
	strategy, err := h.db.GetStrategyByID(id)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve strategy")
		return
	}
	if strategy == nil {
		utils.ErrorResponse(w, http.StatusNotFound, "Strategy not found")
		return
	}
	if strategy.UserID != userID {
		utils.ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := h.db.UpdateStrategyStatus(id, req.Status); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to update strategy status")
		return
	}

	utils.SuccessResponse(w, "Strategy status updated", nil)
}

// GetBacktestResults retrieves backtest results for a strategy
func (h *StrategyHandler) GetBacktestResults(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	idStr := r.URL.Query().Get("strategy_id")
	if idStr == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Strategy ID is required")
		return
	}

	strategyID, err := strconv.Atoi(idStr)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid strategy ID")
		return
	}

	// Verify ownership
	strategy, err := h.db.GetStrategyByID(strategyID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve strategy")
		return
	}
	if strategy == nil {
		utils.ErrorResponse(w, http.StatusNotFound, "Strategy not found")
		return
	}
	if strategy.UserID != userID {
		utils.ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	results, err := h.db.GetBacktestResultsByStrategyID(strategyID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve backtest results")
		return
	}

	utils.SuccessResponse(w, "Backtest results retrieved", results)
}
