package websocket // Keep the same package for simplicity or create a new 'openalgo' package

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time" // --- Need time for history dates ---
)

// --- OpenAlgo Config (Copied from client.go) ---
var (
	oaURL    = "https://openalgo.mywire.org" // Use internal var name
	oaAPIKey = os.Getenv("OPENALGO_API_KEY") // Reads from environment
)

// --- OpenAlgoClient struct to hold config and methods ---
type OpenAlgoClient struct {
	BaseURL string
	APIKey  string
}

// NewOpenAlgoClient creates a new client for interacting with OpenAlgo API
func NewOpenAlgoClient() *OpenAlgoClient {
	if oaAPIKey == "" {
		log.Println("CRITICAL: OPENALGO_API_KEY environment variable not set. OpenAlgo calls will fail.")
	}
	return &OpenAlgoClient{
		BaseURL: oaURL,
		APIKey:  oaAPIKey,
	}
}

// --- Structs for OpenAlgo /api/v1/quotes ---
type OpenAlgoQuoteRequest struct {
	Apikey   string `json:"apikey"`
	Symbol   string `json:"symbol"`
	Exchange string `json:"exchange"`
}

type OpenAlgoQuoteData struct {
	LTP           float64 `json:"ltp"`
	Change        float64 `json:"chng"`
	ChangePercent float64 `json:"chng_perc"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	Open          float64 `json:"open"`
	PreviousClose float64 `json:"prev_close"`
}

type OpenAlgoQuoteResponse struct {
	Status string            `json:"status"`
	Data   OpenAlgoQuoteData `json:"data"`
	Error  string            `json:"error,omitempty"`
}

// --- Structs for OpenAlgo /api/v1/placesmartorder ---
type OpenAlgoSmartOrderRequest struct {
	Apikey       string  `json:"apikey"`
	Strategy     string  `json:"strategy"`
	Symbol       string  `json:"symbol"`
	Exchange     string  `json:"exchange"`
	Action       string  `json:"action"`
	Pricetype    string  `json:"pricetype"`
	Product      string  `json:"product"`
	Quantity     int     `json:"quantity"`
	PositionSize int     `json:"position_size"`
	Price        float64 `json:"price,omitempty"`
}

type OpenAlgoSmartOrderData struct {
	OrderID string `json:"orderId"`
}

type OpenAlgoSmartOrderResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message,omitempty"`
	Data    OpenAlgoSmartOrderData `json:"data"`
	Error   string                 `json:"error,omitempty"`
}

// --- METHOD: fetchOpenAlgoQuote fetches live quote data from OpenAlgo ---
func (oa *OpenAlgoClient) fetchOpenAlgoQuote(symbol string) (*OpenAlgoQuoteData, error) {
	if oa.APIKey == "" {
		return nil, fmt.Errorf("OpenAlgo API key not configured")
	}

	quotesEndpoint := oa.BaseURL + "/api/v1/quotes"

	requestBody := OpenAlgoQuoteRequest{
		Apikey:   oa.APIKey,
		Symbol:   symbol,
		Exchange: "NSE", // Assuming NSE, make dynamic if needed
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal quote request: %w", err)
	}

	resp, err := http.Post(quotesEndpoint, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("http post failed for quote: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read quote response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp OpenAlgoQuoteResponse
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, errResp.Error)
		}
		if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "html") {
			return nil, fmt.Errorf("api request failed with status %d: received HTML page (potential routing issue or endpoint not found)", resp.StatusCode)
		}
		return nil, fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var quoteResponse OpenAlgoQuoteResponse
	if err := json.Unmarshal(bodyBytes, &quoteResponse); err != nil {
		log.Printf("Failed to decode quote response: %v. Body: %s", err, string(bodyBytes))
		return nil, fmt.Errorf("failed to decode quote response: %w. Body: %s", err, string(bodyBytes))
	}

	if quoteResponse.Status != "success" {
		errMsg := quoteResponse.Error
		if errMsg == "" {
			errMsg = "api reported status: " + quoteResponse.Status
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("no data found for symbol %s", symbol)
		}
		return nil, fmt.Errorf("quote api error: %s", errMsg)
	}

	return &quoteResponse.Data, nil
}

// --- METHOD: placeOpenAlgoSmartOrder places a SMART order via OpenAlgo /api/v1/placesmartorder ---
func (oa *OpenAlgoClient) placeOpenAlgoSmartOrder(orderReq *OpenAlgoSmartOrderRequest) (*OpenAlgoSmartOrderResponse, error) {
	if oa.APIKey == "" {
		return nil, fmt.Errorf("OpenAlgo API key not configured")
	}

	orderEndpoint := oa.BaseURL + "/api/v1/placesmartorder"
	orderReq.Apikey = oa.APIKey // Ensure API key is set

	jsonBody, err := json.Marshal(orderReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal smart order request: %w", err)
	}

	resp, err := http.Post(orderEndpoint, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("http post failed for smart order: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read smart order response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp OpenAlgoSmartOrderResponse
		if json.Unmarshal(bodyBytes, &errResp) == nil && (errResp.Error != "" || errResp.Message != "") {
			errMsg := errResp.Error
			if errMsg == "" {
				errMsg = errResp.Message
			}
			if resp.StatusCode == http.StatusBadRequest {
				return nil, fmt.Errorf("%s", errMsg)
			}
			return nil, fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, errMsg)
		}
		if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "html") {
			return nil, fmt.Errorf("api request failed with status %d: received HTML page (potential routing issue or endpoint not found)", resp.StatusCode)
		}
		return nil, fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var orderResponse OpenAlgoSmartOrderResponse
	if err := json.Unmarshal(bodyBytes, &orderResponse); err != nil {
		log.Printf("Failed to decode smart order response: %v. Body: %s", err, string(bodyBytes))
		return nil, fmt.Errorf("failed to decode smart order response: %w. Body: %s", err, string(bodyBytes))
	}

	if orderResponse.Status != "success" {
		errMsg := orderResponse.Message
		if errMsg == "" {
			errMsg = orderResponse.Error
		}
		if errMsg == "" {
			errMsg = "smart order rejected by OpenAlgo (status: " + orderResponse.Status + ")"
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	return &orderResponse, nil
}

// --- METHOD: evaluatePineCondition evaluates Pine Script-like conditions ---
// --- MODIFIED: Added logic to fetch historical data ---
func (oa *OpenAlgoClient) evaluatePineCondition(condition string, symbol string) (bool, error) {
	log.Printf("Attempting to evaluate condition for %s: %s", symbol, condition)

	// --- Step A: Fetch Data ---
	// Define parameters for fetching history
	interval := "5m" // Default to 5-minute candles for now
	exchange := "NSE" // Default exchange
	endDate := time.Now().Format("2006-01-02") // Today in YYYY-MM-DD format

	// Calculate start date to get roughly 100-200 candles (adjust days as needed)
	// Assuming ~75 5-min candles per trading day, ~3 days should be enough.
	// We might need more for longer period indicators. Let's fetch 5 days for safety.
	startDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02") // 5 days ago

	log.Printf("Fetching %s history for %s (%s to %s)", interval, symbol, startDate, endDate)

	candles, err := oa.fetchOpenAlgoHistory(symbol, exchange, interval, startDate, endDate)
	if err != nil {
		// If history fetch fails, we cannot evaluate the condition
		log.Printf("Error fetching history for %s: %v", symbol, err)
		return false, fmt.Errorf("failed to fetch required market data: %w", err)
	}

	if len(candles) == 0 {
		log.Printf("No historical data found for %s in the specified range.", symbol)
		// Decide how to handle this - treat condition as false? Return error?
		return false, fmt.Errorf("no historical data available to evaluate condition")
	}

	log.Printf("Successfully fetched %d candles for %s", len(candles), symbol)

	// --- TODO Steps B, C, D, E: Implement actual parsing and evaluation logic ---
	// B. Add Indicator Library (go get ...)
	// C. Parse 'condition' string (e.g., identify 'rsi(close, 14)' and '> 70')
	// D. Calculate Indicator (e.g., calculate RSI from 'candles' data)
	// E. Evaluate expression (e.g., check if calculated_rsi > 70)

	log.Printf("Placeholder: Condition evaluation logic not implemented after data fetch. Returning false.")
	return false, nil // Placeholder: always returns false until evaluation is built
}


// --- Structs for OpenAlgo /api/v1/history ---
type OpenAlgoHistoryRequest struct {
	Apikey    string `json:"apikey"`
	Symbol    string `json:"symbol"`
	Exchange  string `json:"exchange"`
	Interval  string `json:"interval"`
	StartDate string `json:"start_date"` // YYYY-MM-DD
	EndDate   string `json:"end_date"`   // YYYY-MM-DD
}

type OpenAlgoCandle struct {
	Timestamp int64   `json:"timestamp"` // Unix timestamp
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    int64   `json:"volume"`
	OI        int64   `json:"oi"`
}

type OpenAlgoHistoryResponse struct {
	Status string           `json:"status"`
	Data   []OpenAlgoCandle `json:"data"` // Array of candles
	Error  string           `json:"error,omitempty"`
}

// METHOD: fetchOpenAlgoHistory fetches historical candle data
func (oa *OpenAlgoClient) fetchOpenAlgoHistory(symbol, exchange, interval, startDate, endDate string) ([]OpenAlgoCandle, error) {
	if oa.APIKey == "" {
		return nil, fmt.Errorf("OpenAlgo API key not configured")
	}

	historyEndpoint := oa.BaseURL + "/api/v1/history"

	requestBody := OpenAlgoHistoryRequest{
		Apikey:    oa.APIKey,
		Symbol:    symbol,
		Exchange:  exchange,
		Interval:  interval,
		StartDate: startDate,
		EndDate:   endDate,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal history request: %w", err)
	}

	resp, err := http.Post(historyEndpoint, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("http post failed for history: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read history response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp OpenAlgoHistoryResponse // Use History struct for error parsing
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, errResp.Error)
		}
		if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "html") {
			// Check specifically for 404 on history endpoint
			if resp.StatusCode == http.StatusNotFound {
				return nil, fmt.Errorf("api endpoint not found (status %d): %s - Check OpenAlgo setup", resp.StatusCode, historyEndpoint)
			}
			return nil, fmt.Errorf("api request failed with status %d: received HTML page (potential endpoint issue)", resp.StatusCode)
		}
		return nil, fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var historyResponse OpenAlgoHistoryResponse
	if err := json.Unmarshal(bodyBytes, &historyResponse); err != nil {
		log.Printf("Failed to decode history response: %v. Body: %s", err, string(bodyBytes))
		return nil, fmt.Errorf("failed to decode history response: %w. Body: %s", err, string(bodyBytes))
	}

	if historyResponse.Status != "success" {
		errMsg := historyResponse.Error
		if errMsg == "" {
			errMsg = "api reported status: " + historyResponse.Status
		}
		return nil, fmt.Errorf("history api error: %s", errMsg)
	}

	if historyResponse.Data == nil {
		// Return empty slice if data is null/missing, not an error
		log.Printf("No history data returned for %s (%s %s-%s)", symbol, interval, startDate, endDate)
		return []OpenAlgoCandle{}, nil
	}

	return historyResponse.Data, nil
}
