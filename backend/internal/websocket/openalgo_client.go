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
	// We might need time if fetching history later
	// "time"
)

// --- OpenAlgo Config (Copied from client.go) ---
var (
	oaURL     = "https://openalgo.mywire.org" // Use internal var name
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

	// Use the correct endpoint confirmed via curl
	orderEndpoint := oa.BaseURL + "/api/v1/placesmartorder"

	// Ensure the API key in the request body is set correctly
	orderReq.Apikey = oa.APIKey

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
				// Return only OpenAlgo's specific message for 400 errors
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

// --- METHOD: Placeholder function for evaluating Pine Script-like conditions ---
func (oa *OpenAlgoClient) evaluatePineCondition(condition string, symbol string) (bool, error) {
	log.Printf("Attempting to evaluate condition for %s: %s", symbol, condition)

	// --- TODO: Implement actual parsing and evaluation logic ---
	// 1. Fetch necessary data (e.g., historical candles for the symbol) using OpenAlgo history endpoint (need to add this function).
	// 2. Parse the 'condition' string.
	// 3. Calculate indicators.
	// 4. Evaluate expression.
	// 5. Return result.

	log.Printf("Placeholder: Condition evaluation not implemented. Returning false.")
	return false, nil // Placeholder
}

// --- NEW: Add function to fetch historical data (needed for Pine Script) ---

// Structs for OpenAlgo /api/v1/history
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
		return []OpenAlgoCandle{}, nil
	}

	return historyResponse.Data, nil
}
