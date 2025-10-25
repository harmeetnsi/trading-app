
package openalgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	client  *http.Client
}

// NewClient creates a new OpenAlgo client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Order represents an order request
type Order struct {
	Symbol        string  `json:"symbol"`
	Exchange      string  `json:"exchange"`
	Action        string  `json:"action"` // BUY, SELL
	Quantity      int     `json:"quantity"`
	Price         float64 `json:"price,omitempty"`
	OrderType     string  `json:"order_type"` // MARKET, LIMIT
	Product       string  `json:"product"`    // MIS, CNC, NRML
	PriceType     string  `json:"pricetype"`  // MARKET, LIMIT
	TriggerPrice  float64 `json:"trigger_price,omitempty"`
}

// OrderResponse represents an order response from OpenAlgo
type OrderResponse struct {
	Status  string `json:"status"`
	OrderID string `json:"orderid"`
	Message string `json:"message"`
}

// Position represents a trading position
type Position struct {
	Symbol       string  `json:"symbol"`
	Exchange     string  `json:"exchange"`
	Quantity     int     `json:"quantity"`
	Product      string  `json:"product"`
	AvgPrice     float64 `json:"averageprice"`
	BuyQuantity  int     `json:"buyquantity"`
	SellQuantity int     `json:"sellquantity"`
	BuyPrice     float64 `json:"buyprice"`
	SellPrice    float64 `json:"sellprice"`
	PnL          float64 `json:"pnl"`
}

// PositionsResponse represents positions response
type PositionsResponse struct {
	Status string     `json:"status"`
	Data   []Position `json:"data"`
}

// Holding represents a holding
type Holding struct {
	Symbol       string  `json:"symbol"`
	Exchange     string  `json:"exchange"`
	Quantity     int     `json:"quantity"`
	Product      string  `json:"product"`
	AvgPrice     float64 `json:"averageprice"`
	LastPrice    float64 `json:"lastprice"`
	PnL          float64 `json:"pnl"`
	PnLPercent   float64 `json:"pnlpercentage"`
}

// HoldingsResponse represents holdings response
type HoldingsResponse struct {
	Status string    `json:"status"`
	Data   []Holding `json:"data"`
}

// Quote represents market quote data
type Quote struct {
	Symbol    string  `json:"symbol"`
	Exchange  string  `json:"exchange"`
	LastPrice float64 `json:"lp"`
	Open      float64 `json:"o"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
	Close     float64 `json:"c"`
	Volume    int     `json:"v"`
}

// QuoteResponse represents quote response
type QuoteResponse struct {
	Status string `json:"status"`
	Data   Quote  `json:"data"`
}

// FundsResponse represents funds/margin data
type FundsResponse struct {
	Status string `json:"status"`
	Data   struct {
		AvailableCash float64 `json:"availablecash"`
		UsedMargin    float64 `json:"usedmargin"`
		Net           float64 `json:"net"`
	} `json:"data"`
}

// PlaceOrder places a new order
func (c *Client) PlaceOrder(order Order) (*OrderResponse, error) {
	endpoint := "/api/v1/placeorder"
	
	jsonData, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}

	resp, err := c.makeRequest("POST", endpoint, jsonData)
	if err != nil {
		return nil, err
	}

	var orderResp OrderResponse
	if err := json.Unmarshal(resp, &orderResp); err != nil {
		return nil, err
	}

	return &orderResp, nil
}

// CancelOrder cancels an existing order
func (c *Client) CancelOrder(orderID string) error {
	endpoint := "/api/v1/cancelorder"
	
	data := map[string]string{"orderid": orderID}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = c.makeRequest("POST", endpoint, jsonData)
	return err
}

// GetPositions retrieves current positions
func (c *Client) GetPositions() ([]Position, error) {
	endpoint := "/api/v1/positions"
	
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var posResp PositionsResponse
	if err := json.Unmarshal(resp, &posResp); err != nil {
		return nil, err
	}

	return posResp.Data, nil
}

// GetHoldings retrieves current holdings
func (c *Client) GetHoldings() ([]Holding, error) {
	endpoint := "/api/v1/holdings"
	
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var holdingsResp HoldingsResponse
	if err := json.Unmarshal(resp, &holdingsResp); err != nil {
		return nil, err
	}

	return holdingsResp.Data, nil
}

// GetQuote retrieves market quote for a symbol
func (c *Client) GetQuote(symbol, exchange string) (*Quote, error) {
	endpoint := fmt.Sprintf("/api/v1/quotes?symbol=%s&exchange=%s", symbol, exchange)
	
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var quoteResp QuoteResponse
	if err := json.Unmarshal(resp, &quoteResp); err != nil {
		return nil, err
	}

	return &quoteResp.Data, nil
}

// GetFunds retrieves account funds/margin information
func (c *Client) GetFunds() (*FundsResponse, error) {
	endpoint := "/api/v1/funds"
	
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var fundsResp FundsResponse
	if err := json.Unmarshal(resp, &fundsResp); err != nil {
		return nil, err
	}

	return &fundsResp, nil
}

// ClosePosition closes an existing position
func (c *Client) ClosePosition(symbol, exchange, product string) error {
	endpoint := "/api/v1/closeposition"
	
	data := map[string]string{
		"symbol":   symbol,
		"exchange": exchange,
		"product":  product,
	}
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = c.makeRequest("POST", endpoint, jsonData)
	return err
}

// GetOrderBook retrieves order book
func (c *Client) GetOrderBook() ([]byte, error) {
	endpoint := "/api/v1/orderbook"
	return c.makeRequest("GET", endpoint, nil)
}

// GetTradeBook retrieves trade book
func (c *Client) GetTradeBook() ([]byte, error) {
	endpoint := "/api/v1/tradebook"
	return c.makeRequest("GET", endpoint, nil)
}

// makeRequest makes an HTTP request to OpenAlgo API
func (c *Client) makeRequest(method, endpoint string, body []byte) ([]byte, error) {
	url := c.BaseURL + endpoint

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	// Make request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}

	return respBody, nil
}

// CalculatePortfolio calculates portfolio metrics
func (c *Client) CalculatePortfolio() (map[string]interface{}, error) {
	positions, err := c.GetPositions()
	if err != nil {
		return nil, err
	}

	holdings, err := c.GetHoldings()
	if err != nil {
		return nil, err
	}

	funds, err := c.GetFunds()
	if err != nil {
		return nil, err
	}

	totalPnL := 0.0
	positionsValue := 0.0

	for _, pos := range positions {
		totalPnL += pos.PnL
		positionsValue += float64(pos.Quantity) * pos.AvgPrice
	}

	for _, hold := range holdings {
		totalPnL += hold.PnL
	}

	portfolio := map[string]interface{}{
		"total_value":      funds.Data.Net,
		"cash":             funds.Data.AvailableCash,
		"positions_value":  positionsValue,
		"today_pnl":        totalPnL,
		"total_pnl":        totalPnL,
		"positions":        positions,
		"holdings":         holdings,
	}

	return portfolio, nil
}
