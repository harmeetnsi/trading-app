package models

import (
	"sync"
	"time"
)

// User represents a user in the system
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	TwoFAEnabled bool      `json:"two_fa_enabled"`
	TwoFASecret  string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    int       `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	FileID    *int      `json:"file_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// File represents an uploaded file
type File struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	FileName      string    `json:"file_name"`
	FileType      string    `json:"file_type"` // "pine_script", "csv", "image", "pdf"
	FilePath      string    `json:"file_path"`
	FileSize      int64     `json:"file_size"`
	ProcessedData string    `json:"processed_data,omitempty"` // JSON string of processed data
	CreatedAt     time.Time `json:"created_at"`
}

// Strategy represents a trading strategy
type Strategy struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FileID      int       `json:"file_id"`
	Code        string    `json:"code"`
	Status      string    `json:"status"` // "active", "paused", "stopped"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BacktestResult represents backtest results for a strategy
type BacktestResult struct {
	ID             int       `json:"id"`
	StrategyID     int       `json:"strategy_id"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	InitialCapital float64   `json:"initial_capital"`
	FinalCapital   float64   `json:"final_capital"`
	TotalReturn    float64   `json:"total_return"`
	TotalTrades    int       `json:"total_trades"`
	WinningTrades  int       `json:"winning_trades"`
	LosingTrades   int       `json:"losing_trades"`
	MaxDrawdown    float64   `json:"max_drawdown"`
	SharpeRatio    float64   `json:"sharpe_ratio"`
	ResultData     string    `json:"result_data"` // JSON string of detailed results
	CreatedAt      time.Time `json:"created_at"`
}

// Trade represents a trade execution
type Trade struct {
	ID         int        `json:"id"`
	UserID     int        `json:"user_id"`
	StrategyID *int       `json:"strategy_id,omitempty"`
	Symbol     string     `json:"symbol"`
	Action     string     `json:"action"` // "BUY", "SELL"
	Quantity   int        `json:"quantity"`
	Price      float64    `json:"price"`
	OrderType  string     `json:"order_type"` // "MARKET", "LIMIT"
	Status     string     `json:"status"`     // "pending", "executed", "failed"
	OrderID    string     `json:"order_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ExecutedAt *time.Time `json:"executed_at,omitempty"`
}

// Position represents an open position
type Position struct {
	Symbol       string    `json:"symbol"`
	Quantity     int       `json:"quantity"`
	AvgPrice     float64   `json:"avg_price"`
	CurrentPrice float64   `json:"current_price"`
	PnL          float64   `json:"pnl"`
	PnLPercent   float64   `json:"pnl_percent"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Portfolio represents overall portfolio metrics
type Portfolio struct {
	TotalValue      float64    `json:"total_value"`
	Cash            float64    `json:"cash"`
	PositionsValue  float64    `json:"positions_value"`
	TodayPnL        float64    `json:"today_pnl"`
	TotalPnL        float64    `json:"total_pnl"`
	TotalPnLPercent float64    `json:"total_pnl_percent"`
	Positions       []Position `json:"positions"`
}

// OpenPosition represents a currently held position in the portfolio
type OpenPosition struct {
	UserID     int       `json:"user_id"`
	Symbol     string    `json:"symbol"`
	Exchange   string    `json:"exchange"`
	Quantity   int       `json:"quantity"`
	EntryPrice float64   `json:"entry_price"`
	CreatedAt  time.Time `json:"created_at"`
}

// OrderState represents the current state of an auto order
type OrderState int

const (
	StateMonitoring OrderState = iota
	StateEvaluating
	StateExecuting
	StateCompleted
	StateFailed
	StateExpired
)

// AutoOrder represents a running background conditional order
type AutoOrder struct {
	ID        string    `json:"id"`        // Unique ID for tracking/cancellation
	UserID    int       `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Exchange  string    `json:"exchange"`
	Product   string    `json:"product"`   // MIS, NRML, CNC
	Quantity  int       `json:"quantity"`
	Action    string    `json:"action"`
	Interval  string    `json:"interval"`
	Condition string    `json:"condition"`
	Status    string    `json:"status"`    // e.g., "running", "executed", "cancelled"
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"` // Defines when monitoring stops
	
	// State management fields
	State       OrderState
	StateMux    sync.RWMutex
	CleanupOnce sync.Once
}