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
	OrderID string `json:"orderid"`
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

type OpenAlgoOrderStatusRequest struct {
	Apikey   string `json:"apikey"`
	Strategy string `json:"strategy"`
	OrderID  string `json:"orderid"`
}

type OpenAlgoOrderStatusData struct {
	Action       string  `json:"action"`
	AveragePrice float64 `json:"average_price"`
	Exchange     string  `json:"exchange"`
	OrderStatus  string  `json:"order_status"`
	OrderID      string  `json:"orderid"`
	Price        float64 `json:"price"`
	PriceType    string  `json:"pricetype"`
	Product      string  `json:"product"`
	Quantity     string  `json:"quantity"`
	Symbol       string  `json:"symbol"`
	Timestamp    string  `json:"timestamp"`
	TriggerPrice float64 `json:"trigger_price"`
}

type OpenAlgoOrderStatusResponse struct {
	Status string                  `json:"status"`
	Data   OpenAlgoOrderStatusData `json:"data"`
	Error  string                  `json:"error,omitempty"`
}

// --- METHOD: FetchOpenAlgoQuote fetches live quote data from OpenAlgo ---
func (oa *OpenAlgoClient) FetchOpenAlgoQuote(symbol, exchange string) (*OpenAlgoQuoteData, error) {
	if oa.APIKey == "" {
		return nil, fmt.Errorf("OpenAlgo API key not configured")
	}

	quotesEndpoint := oa.BaseURL + "/api/v1/quotes"

	requestBody := OpenAlgoQuoteRequest{
		Apikey:   oa.APIKey,
		Symbol:   symbol,
		Exchange: exchange,
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

// --- METHOD: FetchOrderStatus fetches the status of a specific order ---
func (oa *OpenAlgoClient) FetchOrderStatus(orderID, strategy string) (*OpenAlgoOrderStatusData, error) {
	if oa.APIKey == "" {
		return nil, fmt.Errorf("OpenAlgo API key not configured")
	}

	statusEndpoint := oa.BaseURL + "/api/v1/orderstatus"

	requestBody := OpenAlgoOrderStatusRequest{
		Apikey:   oa.APIKey,
		Strategy: strategy,
		OrderID:  orderID,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order status request: %w", err)
	}

	resp, err := http.Post(statusEndpoint, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("http post failed for order status: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read order status response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp OpenAlgoOrderStatusResponse
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("api request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var statusResponse OpenAlgoOrderStatusResponse
	if err := json.Unmarshal(bodyBytes, &statusResponse); err != nil {
		log.Printf("Failed to decode order status response: %v. Body: %s", err, string(bodyBytes))
		return nil, fmt.Errorf("failed to decode order status response: %w. Body: %s", err, string(bodyBytes))
	}

	if statusResponse.Status != "success" {
		errMsg := statusResponse.Error
		if errMsg == "" {
			errMsg = "api reported status: " + statusResponse.Status
		}
		return nil, fmt.Errorf("order status api error: %s", errMsg)
	}

	return &statusResponse.Data, nil
}

// METHOD: fetchOpenAlgoHistory fetches historical candle data
func (oa *OpenAlgoClient) FetchOpenAlgoHistory(symbol, exchange, interval, startDate, endDate string) ([]OpenAlgoCandle, error) {
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
		var errResp OpenAlgoHistoryResponse
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
	requiredLength := period + 1

	if len(closePrices) < requiredLength {
		return 0, fmt.Errorf("not enough history data to calculate %s(%d) (need at least %d, got %d)", indicatorName, period, requiredLength, len(closePrices))
	}

	switch strings.ToUpper(indicatorName) {
	case "RSI":
		rsiResults := talib.Rsi(closePrices, period)
		return rsiResults[len(rsiResults)-1], nil
	case "EMA":
		emaResults := talib.Ema(closePrices, period)
		return emaResults[len(emaResults)-1], nil
	case "SMA":
		smaResults := talib.Sma(closePrices, period)
		return smaResults[len(smaResults)-1], nil
	case "MACD":
		_, _, macdResults := talib.Macd(closePrices, 12, 26, 9)
		return macdResults[len(macdResults)-1], nil
	case "ROC":
		rocResults := talib.Roc(closePrices, period)
		return rocResults[len(rocResults)-1], nil
	case "LRS":
		lrsResults := talib.LinearRegSlope(closePrices, period)
		return lrsResults[len(lrsResults)-1], nil
	default:
		return 0, fmt.Errorf("unsupported indicator: %s", indicatorName)
	}
}

// --- METHOD: EvaluatePineCondition evaluates Pine Script-like conditions ---
func (oa *OpenAlgoClient) EvaluatePineCondition(interval, condition, symbol, exchange string) (bool, map[string]float64, error) {
	log.Printf("Attempting to evaluate condition for %s on %s (%s): %s", symbol, exchange, interval, condition)

	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02")

	log.Printf("Fetching %s history for %s (%s to %s) on exchange %s", interval, symbol, startDate, endDate, exchange)

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

	closePrices := make([]float64, len(candles))
	for i, candle := range candles {
		closePrices[i] = candle.Close
	}
	log.Printf("Data extracted. Ready for indicator calculation using %d points.", len(closePrices))

	reWithPeriod := regexp.MustCompile(`([A-Za-z]+)(\d+)`)
	matchesWithPeriod := reWithPeriod.FindAllStringSubmatch(condition, -1)

	parameters := make(map[string]interface{})

	if len(closePrices) > 0 {
		parameters["CLOSE"] = closePrices[len(closePrices)-1]
		parameters["close"] = closePrices[len(closePrices)-1]
	}

	var indicatorName, periodStr, varName string

	reFunctionStyle := regexp.MustCompile(`(?i)(sma|ema|rsi)\s*\(\s*close\s*,\s*(\d+)\s*\)`)
	functionMatches := reFunctionStyle.FindAllStringSubmatch(condition, -1)

	for _, match := range functionMatches {
		funcName := strings.ToUpper(match[1])
		periodStr := match[2]

		period, periodErr := strconv.Atoi(periodStr)
		if periodErr != nil {
			log.Printf("Error converting period '%s' to int: %v", periodStr, periodErr)
			continue
		}

		indicatorValue, calcErr := oa.CalculateIndicatorValue(funcName, period, closePrices)
		if calcErr != nil {
			log.Printf("Error calculating indicator %s(%d): %v", funcName, period, calcErr)
			return false, nil, calcErr
		}

		oldFunc := match[0]
		condition = strings.ReplaceAll(condition, oldFunc, fmt.Sprintf("%.6f", indicatorValue))

		varName = fmt.Sprintf("%s%d", funcName, period)
		parameters[varName] = float64(indicatorValue)
		log.Printf("Calculated %s: %.2f", varName, indicatorValue)
	}

	for _, match := range matchesWithPeriod {
		indicatorName = match[1]
		periodStr = match[2]
		varName = match[0]

		period, periodErr := strconv.Atoi(periodStr)
		if periodErr != nil {
			log.Printf("Error converting period '%s' to int: %v", periodStr, periodErr)
			return false, nil, fmt.Errorf("invalid period specified for indicator %s", indicatorName)
		}

		indicatorValue, calcErr := oa.CalculateIndicatorValue(indicatorName, period, closePrices)
		if calcErr != nil {
			log.Printf("Error calculating indicator %s: %v", varName, calcErr)
			return false, nil, calcErr
		}

		parameters[varName] = float64(indicatorValue)
		log.Printf("Calculated %s: %.2f", varName, indicatorValue)
	}

	if strings.Contains(strings.ToUpper(condition), "MACD") {
		macdValue, macdErr := oa.CalculateIndicatorValue("MACD", 12, closePrices)
		if macdErr != nil {
			log.Printf("Error calculating standalone MACD: %v", macdErr)
			return false, nil, macdErr
		}

		parameters["MACD"] = float64(macdValue)
		log.Printf("Calculated MACD: %.2f", macdValue)
	}

	reNoPeriod := regexp.MustCompile(`(RSI|EMA|SMA)\s`)
	if reNoPeriod.MatchString(condition) {
		log.Printf("Parsing error: Condition '%s' contains indicator without period.", condition)
		return false, nil, fmt.Errorf("invalid indicator syntax. Did you forget the period? (e.g., use RSI14 instead of RSI)")
	}

	if len(parameters) == 1 && !strings.Contains(strings.ToUpper(condition), "MACD") {
		log.Printf("Warning: No recognized indicators found in condition: %s. Assuming literal evaluation.", condition)
	}

	expression, err := govaluate.NewEvaluableExpression(condition)
	if err != nil {
		log.Printf("Error parsing condition '%s': %v", condition, err)
		return false, nil, fmt.Errorf("invalid Pine Script condition syntax: %w", err)
	}

	result, err := expression.Evaluate(parameters)
	if err != nil {
		log.Printf("Error evaluating condition: %v", err)
		return false, nil, fmt.Errorf("error during condition evaluation. Check your indicator names and syntax. Details: %v", err)
	}

	isConditionMet, ok := result.(bool)
	if !ok {
		log.Printf("Evaluation result not a boolean: %v (Type: %T)", result, result)
		return false, nil, fmt.Errorf("condition must evaluate to TRUE or FALSE (got type %T). Did you forget a comparison operator (>, <, ==, etc.)?", result)
	}

	indicatorValues := make(map[string]float64)
	for name, value := range parameters {
		if floatVal, ok := value.(float64); ok {
			indicatorValues[name] = floatVal
		}
	}

	log.Printf("Evaluation complete. Condition met: %t", isConditionMet)
	return isConditionMet, indicatorValues, nil
}
