package openalgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/markcheno/go-talib"
)

// --- OpenAlgoClient struct to hold config and methods ---
type OpenAlgoClient struct {
	BaseURL string
	APIKey  string
}

// NewOpenAlgoClient creates a new client for interacting with OpenAlgo API
func NewOpenAlgoClient(baseURL, apiKey string) *OpenAlgoClient {
	if apiKey == "" {
		log.Println("CRITICAL: OpenAlgo API key not configured. OpenAlgo calls will fail.")
	}
	return &OpenAlgoClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}
}

// --- Structs for OpenAlgo API Calls (Quotes, Orders, History) ---
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

// --- METHOD: FetchOpenAlgoQuote fetches live quote data from OpenAlgo ---
func (oa *OpenAlgoClient) FetchOpenAlgoQuote(symbol string) (*OpenAlgoQuoteData, error) {
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

// --- METHOD: PlaceOpenAlgoSmartOrder places a SMART order via OpenAlgo /api/v1/placesmartorder ---
func (oa *OpenAlgoClient) PlaceOpenAlgoSmartOrder(orderReq *OpenAlgoSmartOrderRequest) (*OpenAlgoSmartOrderResponse, error) {
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

// METHOD: fetchOpenAlgoHistory fetches historical candle data
func (oa *OpenAlgoClient) FetchOpenAlgoHistory(symbol, exchange, interval, startDate, endDate string) ([]OpenAlgoCandle, error) {
	// [History fetching logic is correct and remains here]
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
		log.Printf("No history data returned for %s (%s %s-%s)", symbol, interval, startDate, endDate)
		return []OpenAlgoCandle{}, nil
	}

	return historyResponse.Data, nil
}

// --- METHOD: EvaluatePineCondition evaluates Pine Script-like conditions ---
func (oa *OpenAlgoClient) EvaluatePineCondition(condition string, symbol string) (bool, error) {
	log.Printf("Attempting to evaluate condition for %s: %s", symbol, condition)

	// --- Step A: Fetch Data (Logic already executed) ---
	interval := "5m" 
	exchange := "NSE"
	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02")

	log.Printf("Fetching %s history for %s (%s to %s)", interval, symbol, startDate, endDate)

	candles, err := oa.FetchOpenAlgoHistory(symbol, exchange, interval, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching history for %s: %v", symbol, err)
		return false, fmt.Errorf("failed to fetch required market data: %w", err)
	}

	if len(candles) == 0 {
		log.Printf("No historical data found for %s in the specified range.", symbol)
		return false, fmt.Errorf("no historical data available to evaluate condition")
	}

	log.Printf("Successfully fetched %d candles for %s", len(candles), symbol)

	// --- Step C: Prepare Data for Indicator Calculation ---
	closePrices := make([]float64, len(candles))
	for i, candle := range candles {
		closePrices[i] = candle.Close
	}
	log.Printf("Data extracted. Ready for indicator calculation using %d points.", len(closePrices))

	// --- Step D: Calculate Indicator (RSI 14) ---
	requiredLength := 15
	if len(closePrices) < requiredLength {
		log.Printf("Warning: Not enough data points (%d) for RSI(%d).", len(closePrices), requiredLength)
		return false, fmt.Errorf("not enough history data to calculate required indicator (need %d, got %d)", requiredLength, len(closePrices))
	}

	// Calculate RSI with a period of 14
	rsiResults := talib.Rsi(closePrices, 14)
	latestRSI := rsiResults[len(rsiResults)-1]

	log.Printf("Calculated RSI (14) for %s: %.2f", symbol, latestRSI)

	// --- Step E: Implement Parsing and Evaluation (New Logic) ---
	// 1. Create parameter map to hold calculated values
	parameters := make(map[string]interface{})
	parameters["RSI"] = latestRSI
	
	// 2. Parse the condition string (e.g., "(RSI < 30)")
	expression, err := govaluate.NewEvaluableExpression(condition)
	if err != nil {
		log.Printf("Error parsing condition '%s': %v", condition, err)
		return false, fmt.Errorf("invalid Pine Script condition: %w", err)
	}

	// 3. Evaluate the expression
	result, err := expression.Evaluate(parameters)
	if err != nil {
		log.Printf("Error evaluating condition: %v", err)
		return false, fmt.Errorf("error during evaluation: %w", err)
	}

	// 4. Cast the result to a boolean
	isConditionMet, ok := result.(bool)
	if !ok {
		// This happens if the user enters a mathematical expression that doesn't resolve to true/false (e.g., "RSI + 5")
		log.Printf("Evaluation result not a boolean: %v", result)
		return false, fmt.Errorf("condition must evaluate to true or false, got %v", result)
	}

	log.Printf("Evaluation complete. Condition met: %t", isConditionMet)
	return isConditionMet, nil
}
