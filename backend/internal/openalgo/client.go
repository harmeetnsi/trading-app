package openalgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
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
	Status string          `json:"status"`
	Data   OpenAlgoQuoteData `json:"data"`
	Error  string          `json:"error,omitempty"`
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
// NOTE: Now accepts 'exchange' argument
func (oa *OpenAlgoClient) FetchOpenAlgoQuote(symbol, exchange string) (*OpenAlgoQuoteData, error) {
	if oa.APIKey == "" {
		return nil, fmt.Errorf("OpenAlgo API key not configured")
	}

	quotesEndpoint := oa.BaseURL + "/api/v1/quotes"

	requestBody := OpenAlgoQuoteRequest{
		Apikey:   oa.APIKey,
		Symbol:   symbol,
		Exchange: exchange, // <-- Now uses the dynamic 'exchange' argument
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
			errMsg = fmt.Sprintf("no data found for symbol %s on exchange %s", symbol, exchange)
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


// --- NEW METHOD: CalculateIndicatorValue calculates the latest value for a given indicator and period. ---
func (oa *OpenAlgoClient) CalculateIndicatorValue(indicatorName string, period int, closePrices []float64) (float64, error) {
	// Most TA libraries need at least the period for an initial calculation
	requiredLength := period + 1 

	if len(closePrices) < requiredLength {
		return 0, fmt.Errorf("not enough history data to calculate %s(%d) (need at least %d, got %d)", indicatorName, period, requiredLength, len(closePrices))
	}

	switch strings.ToUpper(indicatorName) {
	case "RSI":
		// Calculate RSI
		rsiResults := talib.Rsi(closePrices, period)
		return rsiResults[len(rsiResults)-1], nil

	case "EMA":
		// Calculate EMA
		emaResults := talib.Ema(closePrices, period)
		return emaResults[len(emaResults)-1], nil

	case "MACD":
		// Calculate MACD (using standard 12, 26, 9 periods for simplicity)
		_, _, macdResults := talib.Macd(closePrices, 12, 26, 9) 
		return macdResults[len(macdResults)-1], nil

	case "ROC": // Rate of Change (Momentum)
		// Calculate Rate of Change over the specified period.
		rocResults := talib.Roc(closePrices, period)
		return rocResults[len(rocResults)-1], nil
	
	case "LRS": // Linear Regression Slope (The slope of the trend line)
		// Calculate the slope of the linear regression line over the period.
		lrsResults := talib.LinearRegSlope(closePrices, period)
		return lrsResults[len(lrsResults)-1], nil

	default:
		return 0, fmt.Errorf("unsupported indicator: %s", indicatorName)
	}
}

// --- METHOD: EvaluatePineCondition evaluates Pine Script-like conditions ---
// NOTE: Now accepts 'exchange' argument
// --- METHOD: EvaluatePineCondition evaluates Pine Script-like conditions ---
func (oa *OpenAlgoClient) EvaluatePineCondition(condition string, symbol string, exchange string) (bool, map[string]float64, error) {
	log.Printf("Attempting to evaluate condition for %s on %s: %s", symbol, exchange, condition)

	// --- Step A: Fetch Data (Logic for 5m interval) ---
	interval := "5m"
	endDate := time.Now().Format("2006-01-02")
	// Use 5 days for data fetching to ensure we have enough candles for high periods
	startDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02") 

	log.Printf("Fetching %s history for %s (%s to %s) on exchange %s", interval, symbol, startDate, endDate, exchange)

	// Note: exchange is passed dynamically here
	candles, err := oa.FetchOpenAlgoHistory(symbol, exchange, interval, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching history for %s: %v", symbol, err)
		return false, nil, fmt.Errorf("failed to fetch required market data: %w", err)
	}

	if len(candles) == 0 {
		log.Printf("No historical data found for %s on exchange %s in the specified range.", symbol, exchange)
		return false, nil, fmt.Errorf("no historical data available to evaluate condition")
	}

	log.Printf("Successfully fetched %d candles for %s on exchange %s", len(candles), symbol, exchange)

	// --- Step B: Prepare Data for Indicator Calculation ---
	closePrices := make([]float64, len(candles))
	for i, candle := range candles {
		closePrices[i] = candle.Close
	}
	log.Printf("Data extracted. Ready for indicator calculation using %d points.", len(closePrices))

	// --- Step C: Extract Indicators from the Condition ---
	reWithPeriod := regexp.MustCompile(`([A-Za-z]+)(\d+)`)
	matchesWithPeriod := reWithPeriod.FindAllStringSubmatch(condition, -1)
	
	parameters := make(map[string]interface{}) // Required by govaluate

	// 1. Handle Indicators WITH Periods (RSI14, EMA20)
	for _, match := range matchesWithPeriod {
		indicatorName := match[1] // e.g., "RSI"
		periodStr := match[2]    // e.g., "14"
		varName := match[0]      // e.g., "RSI14"

		period, err := strconv.Atoi(periodStr)
		if err != nil {
			log.Printf("Error converting period '%s' to int: %v", periodStr, err)
			return false, nil, fmt.Errorf("invalid period specified for indicator %s", indicatorName)
		}

		value, err := oa.CalculateIndicatorValue(indicatorName, period, closePrices)
		if err != nil {
			log.Printf("Error calculating indicator %s: %v", varName, err)
			return false, nil, err
		}

		parameters[varName] = float64(value)
		log.Printf("Calculated %s: %.2f", varName, value)
	}

	// 2. Handle Standalone Indicator MACD
	if strings.Contains(strings.ToUpper(condition), "MACD") {
		value, err := oa.CalculateIndicatorValue("MACD", 12, closePrices) // Period (12) is arbitrary for MACD
		if err != nil {
			log.Printf("Error calculating standalone MACD: %v", err)
			return false, nil, err
		}
		
		parameters["MACD"] = float64(value)
		log.Printf("Calculated MACD: %.2f", value)
	}
	
	// Safety check for other custom errors (like 'RSI' without period)
	reNoPeriod := regexp.MustCompile(`(RSI|EMA|SMA)\s`)
	if reNoPeriod.MatchString(condition) {
		log.Printf("Parsing error: Condition '%s' contains indicator without period.", condition)
		return false, nil, fmt.Errorf("invalid indicator syntax. Did you forget the period? (e.g., use RSI14 instead of RSI)")
	}
	
	// Check if any indicators were found. 
	if len(parameters) == 0 && !strings.Contains(strings.ToUpper(condition), "MACD") {
		log.Printf("Warning: No custom indicators found in condition: %s. Assuming literal evaluation.", condition)
	}
	
	// --- Step D: Implement Parsing and Evaluation ---
	
	// 1. Parse the condition string
	expression, err := govaluate.NewEvaluableExpression(condition)
	if err != nil {
		log.Printf("Error parsing condition '%s': %v", condition, err)
		return false, nil, fmt.Errorf("invalid Pine Script condition syntax: %w", err)
	}

	// 2. Evaluate the expression
	result, err := expression.Evaluate(parameters)
	if err != nil {
		log.Printf("Error evaluating condition: %v", err)
		return false, nil, fmt.Errorf("error during condition evaluation. Check your indicator names and syntax. Details: %v", err)
	}

	// 3. Cast the result to a boolean
	isConditionMet, ok := result.(bool)
	if !ok {
		log.Printf("Evaluation result not a boolean: %v (Type: %T)", result, result)
		return false, nil, fmt.Errorf("condition must evaluate to TRUE or FALSE (got type %T). Did you forget a comparison operator (>, <, ==, etc.)?", result)
	}
	
	// --- CONVERT map[string]interface{} TO map[string]float64 for return ---
	indicatorValues := make(map[string]float64)
	for name, value := range parameters {
	    if floatVal, ok := value.(float64); ok {
	        indicatorValues[name] = floatVal
	    }
	}
	// --- END CONVERSION ---

	log.Printf("Evaluation complete. Condition met: %t", isConditionMet)
	return isConditionMet, indicatorValues, nil 
}