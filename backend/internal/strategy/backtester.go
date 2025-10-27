package strategy

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"trading-app/internal/database"
	"trading-app/internal/models"
	"trading-app/internal/openalgo"
)

// Backtester runs backtests on trading strategies
type Backtester struct {
	db *database.DB
	openalgo *openalgo.OpenAlgoClient // CORRECTED: Changed 'Client' to 'OpenAlgoClient'
}

// NewBacktester creates a new backtester
func NewBacktester(db *database.DB, openalgoClient *openalgo.OpenAlgoClient) *Backtester { // CORRECTED: Changed 'Client' to 'OpenAlgoClient'
	return &Backtester{
		db: db,
		openalgo: openalgoClient,
	}
}

// BacktestParams represents backtest parameters
type BacktestParams struct {
	StrategyID int `json:"strategy_id"`
	StartDate time.Time `json:"start_date"`
	EndDate time.Time `json:"end_date"`
	InitialCapital float64 `json:"initial_capital"`
	Symbol string `json:"symbol"`
	Exchange string `json:"exchange"`
}

// BacktestTrade represents a trade in the backtest
type BacktestTrade struct {
	Timestamp time.Time `json:"timestamp"`
	Action string `json:"action"`
	Price float64 `json:"price"`
	Quantity int `json:"quantity"`
	PnL float64 `json:"pnl"`
}

// BacktestMetrics contains detailed backtest metrics
type BacktestMetrics struct {
	Trades []BacktestTrade `json:"trades"`
	EquityCurve []float64 `json:"equity_curve"`
	DrawdownCurve []float64 `json:"drawdown_curve"`
}

// RunBacktest executes a backtest
func (b *Backtester) RunBacktest(params BacktestParams) (*models.BacktestResult, error) {
	// Get strategy
	strategy, err := b.db.GetStrategyByID(params.StrategyID)
	if err != nil {
		return nil, err
	}
	if strategy == nil {
		return nil, fmt.Errorf("strategy not found")
	}

	// For this MVP, we'll create a simplified backtest
	// In production, you would:
	// 1. Fetch historical data from OpenAlgo/broker
	// 2. Parse and execute the strategy code
	// 3. Simulate trades based on strategy signals

	// Simplified simulation
	trades, metrics := b.simulateStrategy(params)

	// Calculate metrics
	totalReturn := metrics.EquityCurve[len(metrics.EquityCurve)-1] - params.InitialCapital
	returnPercent := (totalReturn / params.InitialCapital) * 100

	winningTrades := 0
	losingTrades := 0
	for _, trade := range trades {
		if trade.PnL > 0 {
			winningTrades++
		} else if trade.PnL < 0 {
			losingTrades++
		}
	}

	maxDrawdown := b.calculateMaxDrawdown(metrics.DrawdownCurve)
	sharpeRatio := b.calculateSharpeRatio(metrics.EquityCurve, params.InitialCapital)

	// Serialize metrics
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return nil, err
	}

	// Create backtest result
	result := &models.BacktestResult{
		StrategyID: params.StrategyID,
		StartDate: params.StartDate,
		EndDate: params.EndDate,
		InitialCapital: params.InitialCapital,
		FinalCapital: metrics.EquityCurve[len(metrics.EquityCurve)-1],
		TotalReturn: returnPercent,
		TotalTrades: len(trades),
		WinningTrades: winningTrades,
		LosingTrades: losingTrades,
		MaxDrawdown: maxDrawdown,
		SharpeRatio: sharpeRatio,
		ResultData: string(metricsJSON),
	}

	// Save to database
	savedResult, err := b.db.CreateBacktestResult(result)
	if err != nil {
		return nil, err
	}

	return savedResult, nil
}

// simulateStrategy simulates strategy execution (simplified)
func (b *Backtester) simulateStrategy(params BacktestParams) ([]BacktestTrade, BacktestMetrics) {
	// This is a simplified simulation
	// In production, you would fetch real historical data and execute strategy logic

	trades := []BacktestTrade{}
	equityCurve := []float64{params.InitialCapital}
	drawdownCurve := []float64{0}

	currentCapital := params.InitialCapital
	position := 0
	entryPrice := 0.0

	// Simulate 50 days of trading
	days := int(params.EndDate.Sub(params.StartDate).Hours() / 24)
	if days > 100 {
		days = 100 // Limit for demo
	}

	// Generate random trades for demo
	// In production, this would be based on actual strategy signals
	for i := 0; i < days; i++ {
		timestamp := params.StartDate.Add(time.Duration(i) * 24 * time.Hour)

		// Simulate price movement (random walk)
		price := 100.0 + float64(i)*0.5 + (float64(i%10) - 5)

		// Simple strategy: buy if no position, sell if in position
		if i%5 == 0 {
			if position == 0 {
				// Buy
				quantity := int(currentCapital * 0.2 / price) // Use 20% of capital
				if quantity > 0 {
					position = quantity
					entryPrice = price
					currentCapital -= float64(quantity) * price

					trades = append(trades, BacktestTrade{
						Timestamp: timestamp,
						Action: "BUY",
						Price: price,
						Quantity: quantity,
						PnL: 0,
					})
				}
			} else {
				// Sell
				pnl := (price - entryPrice) * float64(position)
				currentCapital += float64(position) * price

				trades = append(trades, BacktestTrade{
					Timestamp: timestamp,
					Action: "SELL",
					Price: price,
					Quantity: position,
					PnL: pnl,
				})

				position = 0
				entryPrice = 0
			}
		}

		// Calculate current equity
		equity := currentCapital
		if position > 0 {
			equity += float64(position) * price
		}

		equityCurve = append(equityCurve, equity)

		// Calculate drawdown
		maxEquity := equityCurve[0]
		for _, e := range equityCurve {
			if e > maxEquity {
				maxEquity = e
			}
		}
		drawdown := ((maxEquity - equity) / maxEquity) * 100
		drawdownCurve = append(drawdownCurve, drawdown)
	}

	metrics := BacktestMetrics{
		Trades: trades,
		EquityCurve: equityCurve,
		DrawdownCurve: drawdownCurve,
	}

	return trades, metrics
}

// calculateMaxDrawdown calculates maximum drawdown
func (b *Backtester) calculateMaxDrawdown(drawdownCurve []float64) float64 {
	maxDD := 0.0
	for _, dd := range drawdownCurve {
		if dd > maxDD {
			maxDD = dd
		}
	}
	return maxDD
}

// calculateSharpeRatio calculates Sharpe ratio
func (b *Backtester) calculateSharpeRatio(equityCurve []float64, initialCapital float64) float64 {
	if len(equityCurve) < 2 {
		return 0
	}

	// Calculate daily returns
	returns := []float64{}
	for i := 1; i < len(equityCurve); i++ {
		dailyReturn := (equityCurve[i] - equityCurve[i-1]) / equityCurve[i-1]
		returns = append(returns, dailyReturn)
	}

	// Calculate mean return
	meanReturn := 0.0
	for _, r := range returns {
		meanReturn += r
	}
	meanReturn /= float64(len(returns))

	// Calculate standard deviation
	variance := 0.0
	for _, r := range returns {
		variance += math.Pow(r-meanReturn, 2)
	}
	variance /= float64(len(returns))
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return 0
	}

	// Sharpe ratio (assuming risk-free rate = 0)
	sharpeRatio := meanReturn / stdDev * math.Sqrt(252) // Annualized

	return sharpeRatio
}
