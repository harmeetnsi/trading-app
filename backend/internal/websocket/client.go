package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"trading-app/internal/ai"
	"trading-app/internal/database"
	"trading-app/internal/email"
	"trading-app/internal/models"
	"trading-app/internal/openalgo"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

type Client struct {
	hub            *Hub
	conn           *websocket.Conn
	send           chan []byte
	userID         int
	db             *database.DB
	ai             *ai.AIClient
	oaClient       *openalgo.OpenAlgoClient
	autoOrders     map[string]*models.AutoOrder
	orderMux       sync.Mutex
	cancellation   map[string]chan struct{}
	emailService   *email.EmailService
	emailRecipient string
}

type Message struct {
	Type    string      `json:"type"`
	Content string      `json:"content,omitempty"`
	FileID  *int        `json:"file_id,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func NewClient(hub *Hub, conn *websocket.Conn, userID int, db *database.DB, aiClient *ai.AIClient, baseURL string, apiKey string, emailService *email.EmailService, emailRecipient string) *Client {
	return &Client{
		hub:            hub,
		conn:           conn,
		send:           make(chan []byte, 256),
		userID:         userID,
		db:             db,
		ai:             aiClient,
		oaClient:       openalgo.NewOpenAlgoClient(baseURL, apiKey),
		autoOrders:     make(map[string]*models.AutoOrder),
		cancellation:   make(map[string]chan struct{}),
		emailService:   emailService,
		emailRecipient: emailRecipient,
	}
}

func (c *Client) StartAutoOrderMonitoring(symbol, exchange, product, interval, condition, action string, quantity int, expiresAt time.Time) (string, error) {
	orderID := fmt.Sprintf("SO-%d", time.Now().Unix()%100000)
	cancelChan := make(chan struct{})

	order := &models.AutoOrder{
		ID:        orderID,
		UserID:    c.userID,
		Symbol:    symbol,
		Exchange:  exchange,
		Product:   product,  // NEW
		Quantity:  quantity,
		Action:    action,
		Interval:  interval,
		Condition: condition,
		Status:    "running",
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	c.orderMux.Lock()
	c.autoOrders[orderID] = order
	c.cancellation[orderID] = cancelChan
	c.orderMux.Unlock()

	go c.monitorAndPlaceOrder(order)

	return orderID, nil
}

func (c *Client) sendError(errMsg string) {
	errorMsg := Message{
		Type: "error",
		Data: map[string]string{"message": errMsg},
	}
	errorBytes, err := json.Marshal(errorMsg)
	if err == nil {
		c.send <- errorBytes
	} else {
		log.Printf("Failed to marshal error message: %v", err)
	}
}

// monitorAndPlaceOrder is the worker function that runs in a separate goroutine.
func (c *Client) monitorAndPlaceOrder(order *models.AutoOrder) {
	// NEW: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🚨 PANIC in monitorAndPlaceOrder for %s: %v", order.Symbol, r)
			c.sendError(fmt.Sprintf("❌ Auto-Order %s crashed: %v.", order.ID, r))
			c.emailService.SendEmail(c.emailRecipient, "Auto-Order Process crashed", fmt.Sprintf("Auto-Order %s crashed: %v", order.ID, r))
			if time.Now().Before(order.ExpiresAt) {
				c.sendSystemMessage(fmt.Sprintf(" restarting monitoring for order %s.", order.ID))
				go c.monitorAndPlaceOrder(order)
			} else {
				c.sendSystemMessage(fmt.Sprintf(" order %s has expired and will not be restarted.", order.ID))
				c.removeAutoOrder(order.ID)
			}
		}
	}()

	log.Printf("AUTO-ORDER: Monitoring started for %s on %s. Interval: %s. Condition: %s",
		order.Symbol, order.Exchange, order.Interval, order.Condition)

	// Retrieve the cancellation channel for this specific order
	c.orderMux.Lock()
	cancelChan, ok := c.cancellation[order.ID]
	c.orderMux.Unlock()

	if !ok {
		log.Printf("AUTO-ORDER ERROR: Could not find cancellation channel for order %s. Stopping.", order.ID)
		return
	}

	// Determine the check delay
	checkDelay, _ := ParseIntervalDuration(order.Interval)

	// Safety check: Do not allow checks more frequent than 5 seconds
	if checkDelay < 5*time.Second {
		checkDelay = 5 * time.Second
	}

	ticker := time.NewTicker(checkDelay)
	defer ticker.Stop()

	// FIX: Create expiry timer properly
	expiryDuration := time.Until(order.ExpiresAt)
	if expiryDuration <= 0 {
		c.sendSystemMessage(fmt.Sprintf("⚠️ Auto-Order %s already expired. Stopping.", order.ID))
		return
	}
	if expiryDuration > 30*24*time.Hour {
		expiryDuration = 30 * 24 * time.Hour // Cap at 30 days
	}

	expiryTimer := time.NewTimer(expiryDuration)
	defer expiryTimer.Stop()

	// Cleanup function to run when the loop exits
	defer func() {
		c.removeAutoOrder(order.ID)
		log.Printf("AUTO-ORDER: Monitoring for %s (ID: %s) stopped and cleaned up.", order.Symbol, order.ID)
	}()

	for {
		select {
		case <-cancelChan:
			// Received explicit signal to stop
			c.sendSystemMessage(fmt.Sprintf("❌ Auto-Order %s for %s was CANCELLED.", order.ID, order.Symbol))
			return

		case <-expiryTimer.C:
			// Monitoring period has expired
			c.sendSystemMessage(fmt.Sprintf("🕒 Auto-Order %s for %s has EXPIRED. Monitoring stopped.", order.ID, order.Symbol))
			return

		case <-ticker.C:
			// Time for the next check

			// Safety check for expiry
			if time.Now().After(order.ExpiresAt) {
				c.sendSystemMessage(fmt.Sprintf("🕒 Auto-Order %s for %s has EXPIRED. Monitoring stopped.", order.ID, order.Symbol))
				return
			}

			// Evaluate condition
			isMet, valuesMap, err := c.oaClient.EvaluatePineCondition(order.Interval, order.Condition, order.Symbol, order.Exchange)

			if err != nil {
				log.Printf("AUTO-ORDER: Evaluation error for %s: %v", order.ID, err)
				// Don't stop monitoring on transient errors - just log and continue
				continue
			}

			if isMet {
				// Build indicator summary
				indicatorSummary := ""
				for name, value := range valuesMap {
					if math.IsNaN(value) || math.IsInf(value, 0) {
						indicatorSummary += fmt.Sprintf(" **%s**: N/A |", name)
					} else {
						indicatorSummary += fmt.Sprintf(" **%s**: %.2f |", name, value)
					}
				}

				// Place order
				orderReq := &openalgo.OpenAlgoSmartOrderRequest{
					Strategy:     "auto_chat",
					Symbol:       order.Symbol,
					Exchange:     order.Exchange,
					Action:       order.Action,
					Pricetype:    "MARKET",
					Product:      order.Product,
					Quantity:     order.Quantity,
					PositionSize: 0,
					Price:        0.0,
				}

				log.Printf("AUTO-ORDER: Placing order for %s (ID: %s)", order.Symbol, order.ID)
				orderResponse, err := c.oaClient.PlaceOpenAlgoSmartOrder(orderReq)

				if err != nil {
					c.sendError(fmt.Sprintf("❌ Auto-Order %s FAILED to place order: %v. Monitoring continues.", order.ID, err))
					c.emailService.SendEmail(c.emailRecipient, "Auto-Order Execution Failed", fmt.Sprintf("Auto-Order %s failed to place order: %v", order.ID, err))
				} else {
					c.sendSystemMessage(fmt.Sprintf("✅ **AUTO ORDER EXECUTED** for %s on %s!\n\n### Trigger Values:\n%s\n**Order ID**: %s\n\nMonitoring continues.",
						order.Symbol, order.Exchange, indicatorSummary, orderResponse.Data.OrderID))
					c.emailService.SendEmail(c.emailRecipient, "Auto-Order Executed", fmt.Sprintf("Auto-Order %s executed for %s on %s.", order.ID, order.Symbol, order.Exchange))
					go c.pollOrderStatus(order.ID, orderResponse.Data.OrderID)
				}
			}
		}
	}
}

// pollOrderStatus checks the status of a submitted order and notifies on failure.
func (c *Client) pollOrderStatus(autoOrderID, brokerOrderID string) {
	const maxRetries = 5
	const retryInterval = 15 * time.Second

	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryInterval)

		// Ensure the original auto-order still exists before checking status
		c.orderMux.Lock()
		autoOrder, exists := c.autoOrders[autoOrderID]
		c.orderMux.Unlock()
		if !exists {
			log.Printf("Order status polling for %s stopped as the auto-order no longer exists.", autoOrderID)
			return
		}

		status, err := c.oaClient.FetchOrderStatus(brokerOrderID, "auto_chat")
		if err != nil {
			log.Printf("Error fetching order status for %s: %v", brokerOrderID, err)
			continue // Don't notify on temporary fetch errors
		}

		log.Printf("Order %s status for %s (%s): %s", brokerOrderID, autoOrder.Symbol, autoOrder.Action, status.OrderStatus)

		switch strings.ToLower(status.OrderStatus) {
		case "complete":
			// Order executed successfully, no need to poll further.
			return
		case "rejected", "cancelled":
			// Terminal failure states, notify the user.
			failureMsg := fmt.Sprintf(
				"⚠️ **Order Failure Notice** ⚠️\n\nYour auto-order for **%s** (%s) with broker ID **%s** was **%s**.",
				autoOrder.Symbol, autoOrder.Action, brokerOrderID, strings.ToUpper(status.OrderStatus),
			)
			c.sendSystemMessage(failureMsg)
			c.emailService.SendEmail(
				c.emailRecipient,
				fmt.Sprintf("Auto-Order %s for %s was %s", autoOrder.ID, autoOrder.Symbol, strings.ToUpper(status.OrderStatus)),
				failureMsg,
			)
			return // Stop polling on terminal status.
		}
	}

	// If the loop finishes without a "complete" or "rejected" status, notify as unresolved.
	c.orderMux.Lock()
	autoOrder, exists := c.autoOrders[autoOrderID]
	c.orderMux.Unlock()
	if !exists {
		return // The original order was cancelled in the meantime.
	}
	unresolvedMsg := fmt.Sprintf(
		"⚠️ **Order Status Unresolved** ⚠️\n\nYour auto-order for **%s** (%s) with broker ID **%s** could not be confirmed as 'complete' after several checks. Please verify its status manually.",
		autoOrder.Symbol, autoOrder.Action, brokerOrderID,
	)
	c.sendSystemMessage(unresolvedMsg)
	c.emailService.SendEmail(
		c.emailRecipient,
		fmt.Sprintf("Auto-Order %s for %s - Status Unresolved", autoOrder.ID, autoOrder.Symbol),
		unresolvedMsg,
	)
}

// removeAutoOrder cleans up the tracking maps after an order is executed, cancelled, or expired.
func (c *Client) removeAutoOrder(orderID string) {
	c.orderMux.Lock()
	order, exists := c.autoOrders[orderID]
	if !exists {
		c.orderMux.Unlock()
		log.Printf("AUTO-ORDER: Order %s already removed", orderID)
		return
	}

	// Use sync.Once to ensure cleanup happens exactly once
	order.CleanupOnce.Do(func() {
		log.Printf("AUTO-ORDER: Cleaning up order %s", orderID)

		// Remove from map first
		delete(c.autoOrders, orderID)

		// Close channel safely
		if ch, ok := c.cancellation[orderID]; ok {
			select {
			case <-ch:
			// Already closed
			default:
				close(ch)
			}
			delete(c.cancellation, orderID)
		}
	})

	c.orderMux.Unlock()
}

func (c *Client) sendSystemMessage(content string) {
	msg := Message{
		Type:    "chat",
		Content: content,
		Data: map[string]interface{}{
			"role":       "system",
			"created_at": time.Now(),
		},
	}
	msgBytes, _ := json.Marshal(msg)
	c.send <- msgBytes
}

func ParseIntervalDuration(interval string) (time.Duration, error) {
	switch strings.ToLower(interval) {
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	default:
		d, err := time.ParseDuration(interval)
		if err != nil {
			return 0, fmt.Errorf("invalid or unsupported interval format: %s", interval)
		}
		return d, nil
	}
}

func parseValidity(validityStr string) (time.Time, error) {
	if strings.ToLower(validityStr) == "forever" {
		return time.Date(9999, time.January, 1, 0, 0, 0, 0, time.UTC), nil
	}

	duration, err := time.ParseDuration(validityStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid duration format")
	}

	if duration > 30*24*time.Hour {
		return time.Time{}, fmt.Errorf("maximum validity period is 30 days")
	}

	return time.Now().Add(duration), nil
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	c.conn.SetReadLimit(maxMessageSize)

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		switch msg.Type {
		case "chat":
			c.handleChatMessage(&msg)
		case "typing":
			c.send <- messageBytes
		case "ping":
			pong := Message{Type: "pong"}
			pongBytes, _ := json.Marshal(pong)
			c.send <- pongBytes
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleChatMessage(msg *Message) {
	chatMsg := &models.ChatMessage{
		UserID:  c.userID,
		Role:    "user",
		Content: msg.Content,
		FileID:  msg.FileID,
	}

	savedMsg, err := c.db.CreateChatMessage(chatMsg)
	if err != nil {
		log.Printf("Failed to save chat message: %v", err)
		c.sendError("Failed to save message")
		return
	}

	userMsgResponse := Message{
		Type:    "chat",
		Content: savedMsg.Content,
		Data: map[string]interface{}{
			"id":         savedMsg.ID,
			"role":       "user",
			"created_at": savedMsg.CreatedAt,
			"file_id":    savedMsg.FileID,
		},
	}
	userMsgBytes, _ := json.Marshal(userMsgResponse)
	c.send <- userMsgBytes

	if strings.HasPrefix(msg.Content, "/") {
		go c.handleTradingCommand(msg.Content)
	} else {
		typingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": true}}
		typingBytes, _ := json.Marshal(typingMsg)
		c.send <- typingBytes
		go c.processAIResponse(msg.Content, msg.FileID)
	}
}

func (c *Client) handleTradingCommand(command string) {
	typingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": true}}
	typingBytes, _ := json.Marshal(typingMsg)
	c.send <- typingBytes

	var responseContent string

	parts := strings.Fields(command)
	if len(parts) < 1 {
		responseContent = "Sorry, I didn't understand that command."
	} else {
		cmd := parts[0]
		switch cmd {
		case "/price":
			exchange := "NSE"

			if len(parts) == 3 {
				exchange = strings.ToUpper(parts[2])
			}

			if len(parts) >= 2 {
				symbol := strings.ToUpper(parts[1])
				quote, err := c.oaClient.FetchOpenAlgoQuote(symbol, exchange)
				if err != nil {
					log.Printf("Failed to fetch quote for %s on %s: %v", symbol, exchange, err)
					responseContent = fmt.Sprintf("Sorry, I had trouble fetching the quote for %s on %s: %v", symbol, exchange, err)
				} else {
					responseContent = fmt.Sprintf(
						"Here is the latest price for **%s** on **%s**:\n\n---\n\n### **%s: %s**\n| Metric | Value |\n| :--- | :--- |\n| **Last Traded Price** | **%.2f** |\n| **Change** | **%.2f (%.2f%%)** |\n| **Day's High** | %.2f |\n| **Day's Low** | %.2f |\n| **Open** | %.2f |\n| **Previous Close** | %.2f |\n\n*Data Source: Live feed via OpenAlgo.*",
						symbol, exchange, exchange, symbol, quote.LTP, quote.Change, quote.ChangePercent, quote.High, quote.Low, quote.Open, quote.PreviousClose,
					)
				}
			} else {
				responseContent = "Usage: `/price <SYMBOL> [EXCHANGE]` (e.g., `/price RELIANCE NFO`)"
			}

		case "/buy_smart", "/sell_smart":
			if len(parts) >= 3 {
				action := "BUY"
				if cmd == "/sell_smart" {
					action = "SELL"
				}

				symbol := strings.ToUpper(parts[1])
				quantityStr := parts[2]

				exchange := "NSE"
				priceType := "MARKET"
				product := "MIS"  // Changed here
				price := 0.0

				if len(parts) >= 4 {
					exchange = strings.ToUpper(parts[3])
				}

				if len(parts) >= 5 {
					// Check if it's a product type (MIS/NRML/CNC)
					param := strings.ToUpper(parts[4])
					if param == "MIS" || param == "NRML" || param == "CNC" {
						product = param
						// Check for price type in next parameter
						if len(parts) >= 6 {
							pType := strings.ToUpper(parts[5])
							if pType == "LIMIT" || pType == "MARKET" {
								priceType = pType
							}
						}
						// Handle limit price
						if priceType == "LIMIT" && len(parts) >= 7 {
							priceStr := parts[6]
							p, err := strconv.ParseFloat(priceStr, 64)
							if err != nil {
								responseContent = fmt.Sprintf("Invalid limit price '%s'. Must be a number.", priceStr)
								break
							}
							price = p
						}
					} else if param == "LIMIT" || param == "MARKET" {
						// Old format: price type without product (defaults to MIS)
						priceType = param
						if priceType == "LIMIT" && len(parts) >= 6 {
							priceStr := parts[5]
							p, err := strconv.ParseFloat(priceStr, 64)
							if err != nil {
								responseContent = fmt.Sprintf("Invalid limit price '%s'. Must be a number.", priceStr)
								break
							}
							price = p
						}
					}
				}

				quantity, err := strconv.Atoi(quantityStr)
				if err != nil || quantity <= 0 {
					responseContent = "Invalid quantity. Must be a positive number."
					break
				}

				log.Printf("Placing Smart Order: Action=%s, Symbol=%s, Qty=%d, Exchange=%s, Type=%s, Price=%.2f",
					action, symbol, quantity, exchange, priceType, price)

				orderReq := &openalgo.OpenAlgoSmartOrderRequest{
					Strategy:     "manual_chat",
					Symbol:       symbol,
					Exchange:     exchange,
					Action:       action,
					Pricetype:    priceType,
					Product:      product,  // USE THE VARIABLE
					Quantity:     quantity,
					PositionSize: 0,
					Price:        price,
				}

				orderResponse, err := c.oaClient.PlaceOpenAlgoSmartOrder(orderReq)
				if err != nil {
					log.Printf("Failed to place %s smart order for %s: %v", action, symbol, err)
					responseContent = fmt.Sprintf("❌ Sorry, I had trouble placing your %s smart order: %v", action, err)
				} else {
					priceDisplay := "Market"
					if priceType == "LIMIT" {
						priceDisplay = fmt.Sprintf("Limit @ %.2f", price)
					}
					responseContent = fmt.Sprintf("✅ **Smart Order Submitted!**\n\n- **Action**: %s\n- **Symbol**: %s on %s\n- **Qty**: %d\n- **Price**: %s\n- **Order ID**: %s",
						action, symbol, exchange, quantity, priceDisplay, orderResponse.Data.OrderID)
				}

			} else {
				responseContent = "Usage: `/buy_smart <SYMBOL> <QTY> [EXCHANGE] [MARKET|LIMIT] [PRICE]` or `/sell_smart <SYMBOL> <QTY> [EXCHANGE] [MARKET|LIMIT] [PRICE]`"
			}

		case "/rsi":
			if len(parts) >= 2 {
				symbol := strings.ToUpper(parts[1])
				responseContent = fmt.Sprintf(
					"The simple `/rsi` command is deprecated. Please use the powerful **`/signal`** command for live evaluations, e.g.:\n\n- `/signal %s 5m \"RSI14 < 30\" NSE`\n- `/signal %s 1h \"EMA50 > EMA200\" NFO`",
					symbol, symbol,
				)
			} else {
				responseContent = "Usage: `/rsi <SYMBOL>`. Command deprecated. Use `/signal` instead."
			}

		case "/signal":
			if len(parts) < 4 {
				responseContent = "Usage: `/signal <SYMBOL> <INTERVAL> <CONDITION> [EXCHANGE]`. Example: `/signal RELIANCE 1h \"RSI14 < 30\" NFO`"
				break
			}

			symbol := strings.ToUpper(parts[1])
			interval := strings.ToLower(parts[2])
			exchange := "NSE"

			if interval != "5m" && interval != "15m" && interval != "1h" {
				responseContent = fmt.Sprintf("Unsupported interval: %s. Please use 5m, 15m, or 1h.", interval)
				break
			}

			conditionParts := parts[3:]
			lastPart := strings.ToUpper(conditionParts[len(conditionParts)-1])

			if len(lastPart) >= 2 && len(lastPart) <= 4 && strings.IndexFunc(lastPart, func(r rune) bool {
				return r < 'A' || r > 'Z'
			}) == -1 {
				exchange = lastPart
				conditionParts = conditionParts[:len(conditionParts)-1]
			}

			condition := strings.Join(conditionParts, " ")
			condition = strings.TrimPrefix(condition, "\"")
			condition = strings.TrimSuffix(condition, "\"")

			log.Printf("Received signal command: Symbol=%s, Interval=%s, Condition=\"%s\", Exchange=%s", symbol, interval, condition, exchange)

			isConditionMet, valuesMap, err := c.oaClient.EvaluatePineCondition(interval, condition, symbol, exchange)
			if err != nil {
				log.Printf("Error evaluating condition for %s on %s: %v", symbol, exchange, err)
				responseContent = fmt.Sprintf("⚠️ Error evaluating signal for %s on %s: %v", symbol, exchange, err)
			} else {
				indicatorSummary := ""
				for name, value := range valuesMap {
					indicatorSummary += fmt.Sprintf(" **%s**: %.2f |", name, value)
				}

				if isConditionMet {
					responseContent = fmt.Sprintf("✅ **Signal Met** for %s on %s (%s).\n\n### Current Values:\n%s\nCondition: `%s` is TRUE.", symbol, exchange, interval, indicatorSummary, condition)
				} else {
					responseContent = fmt.Sprintf("❌ **Signal NOT Met** for %s on %s (%s).\n\n### Current Values:\n%s\nCondition: `%s` is FALSE.", symbol, exchange, interval, indicatorSummary, condition)
				}
			}

		case "/buy_smart_auto", "/sell_smart_auto":
			if len(parts) < 8 {  // CHANGED FROM 7 TO 8
				responseContent = "Usage: `/buy_smart_auto <SYMBOL> <QTY> <EXCHANGE> <PRODUCT> <INTERVAL> <VALIDITY> <CONDITION...>`. Example: `/buy_smart_auto TCS 10 NSE NRML 5m 2h \"RSI14 < 30\"`"
				break
			}

			action := "BUY"
			if cmd == "/sell_smart_auto" {
				action = "SELL"
			}

			symbol := strings.ToUpper(parts[1])
			quantityStr := parts[2]
			exchange := strings.ToUpper(parts[3])
			product := strings.ToUpper(parts[4])  // NEW
			interval := strings.ToLower(parts[5])  // SHIFTED
			validityStr := strings.ToLower(parts[6])  // SHIFTED
			condition := strings.Join(parts[7:], " ")  // SHIFTED

			// Validate product
			if product != "MIS" && product != "NRML" && product != "CNC" {
				responseContent = fmt.Sprintf("Invalid product type: %s. Use MIS, NRML, or CNC.", product)
				break
			}

			quantity, err := strconv.Atoi(quantityStr)
			if err != nil || quantity <= 0 {
				responseContent = "Invalid quantity. Must be a positive number."
				break
			}

			if interval != "5m" && interval != "15m" && interval != "1h" {
				responseContent = fmt.Sprintf("Unsupported interval: %s. Please use 5m, 15m, or 1h.", interval)
				break
			}

			expiresAt, err := parseValidity(validityStr)
			if err != nil {
				responseContent = fmt.Sprintf("Invalid validity: %v. Use formats like '2h', '1d', or 'forever'.", err)
				break
			}

			_, initialValues, _ := c.oaClient.EvaluatePineCondition(interval, condition, symbol, exchange)
			indicatorSummary := ""
			for name, value := range initialValues {
				indicatorSummary += fmt.Sprintf(" **%s**: %.2f |", name, value)
			}

			orderID, err := c.StartAutoOrderMonitoring(symbol, exchange, product, interval, condition, action, quantity, expiresAt)

			if err != nil {
				log.Printf("Failed to start auto order for %s: %v", symbol, err)
				responseContent = fmt.Sprintf("❌ Failed to start auto order monitoring: %v", err)
			} else {
				expiryDisplay := "Running Indefinitely"
				if validityStr != "forever" {
					expiryDisplay = fmt.Sprintf("Expires at %s", expiresAt.Format("15:04:05 MST"))
				}

				responseContent = fmt.Sprintf("✅ **Auto Order Monitoring Started!**\n\n### Initial Values:\n%s\n- **ID**: %s\n- **Action**: %s\n- **Symbol**: %s on %s\n- **Interval**: %s\n- **Condition**: `%s`\n- **Validity**: %s",
					indicatorSummary, orderID, action, symbol, exchange, interval, condition, expiryDisplay)
			}

		case "/status_orders":
			c.orderMux.Lock()
			orderCount := len(c.autoOrders)
			if orderCount == 0 {
				responseContent = "📋 **No Active Auto-Orders**\n\nYou currently have no running auto-orders."
				c.orderMux.Unlock()
			} else {
				var orderList strings.Builder
				orderList.WriteString(fmt.Sprintf("📋 **Active Auto-Orders** (%d)\n\n", orderCount))

				for _, order := range c.autoOrders {
					expiryInfo := "Never"
					if order.ExpiresAt.Year() != 9999 {
						expiryInfo = order.ExpiresAt.Format("15:04:05 MST")
					}
					orderList.WriteString(fmt.Sprintf("**ID**: `%s`\n- **Symbol**: %s on %s\n- **Action**: %s\n- **Qty**: %d\n- **Condition**: `%s`\n- **Interval**: %s\n- **Expires**: %s\n- **Status**: %s\n\n",
						order.ID, order.Symbol, order.Exchange, order.Action, order.Quantity, order.Condition, order.Interval, expiryInfo, order.Status))
				}
				responseContent = orderList.String()
				c.orderMux.Unlock()
			}

		case "/cancel_order":
			if len(parts) != 2 {
				responseContent = "Usage: `/cancel_order <ORDER_ID>`"
				break
			}
			orderID := parts[1]

			c.orderMux.Lock()
			order, found := c.autoOrders[orderID]
			cancelChan, chanFound := c.cancellation[orderID]
			c.orderMux.Unlock()

			if !found || order.UserID != c.userID {
				responseContent = fmt.Sprintf("❌ Auto-Order ID `%s` not found or does not belong to you.", orderID)
			} else {
				if chanFound {
					select {
					case cancelChan <- struct{}{}:
					default:
					}
				}
				c.removeAutoOrder(orderID)
				responseContent = fmt.Sprintf("✅ Auto-Order ID `%s` for %s has been successfully cancelled.", orderID, order.Symbol)
			}

		case "/cancel_all_orders":
			count := 0

			c.orderMux.Lock()
			ordersToCancel := []*models.AutoOrder{}
			for _, order := range c.autoOrders {
				if order.UserID == c.userID && order.Status == "running" {
					ordersToCancel = append(ordersToCancel, order)
				}
			}
			c.orderMux.Unlock()

			for _, order := range ordersToCancel {
				c.orderMux.Lock()
				if chanToClose, ok := c.cancellation[order.ID]; ok {
					select {
					case chanToClose <- struct{}{}:
					default:
					}
					c.removeAutoOrder(order.ID)
					count++
				}
				c.orderMux.Unlock()
			}

			responseContent = fmt.Sprintf("✅ Successfully cancelled %d active conditional orders.", count)

		default:
			responseContent = "Sorry, I didn't understand that command. Try: \n- `/price <SYMBOL> [EXCHANGE]`\n- `/buy_smart <SYMBOL> <QTY> [EXCHANGE] [MARKET|LIMIT] [PRICE]`\n- `/signal <SYMBOL> <INTERVAL> <CONDITION> [EXCHANGE]`\n- `/buy_smart_auto <SYMBOL> <QTY> <EXCHANGE> <INTERVAL> <VALIDITY> <CONDITION>`\n- `/status_orders` or `/cancel_order <ID>`"
		}
	}

	assistMsg := &models.ChatMessage{
		UserID:  c.userID,
		Role:    "assistant",
		Content: responseContent,
	}
	var savedAssistMsg *models.ChatMessage
	savedAssistMsg, err := c.db.CreateChatMessage(assistMsg)
	if err != nil {
		log.Printf("Failed to save command response: %v", err)
		savedAssistMsg = &models.ChatMessage{ID: 0, CreatedAt: time.Now()}
	}

	stopTypingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": false}}
	stopTypingBytes, _ := json.Marshal(stopTypingMsg)
	c.send <- stopTypingBytes

	assistMsgResponse := Message{
		Type:    "chat",
		Content: responseContent,
		Data: map[string]interface{}{
			"id":         savedAssistMsg.ID,
			"role":       "assistant",
			"created_at": savedAssistMsg.CreatedAt,
		},
	}
	assistMsgBytes, _ := json.Marshal(assistMsgResponse)
	c.send <- assistMsgBytes
}

func (c *Client) processAIResponse(userMessage string, fileID *int) {
	history, err := c.db.GetChatMessagesByUserID(c.userID, 10)
	if err != nil {
		log.Printf("Failed to get chat history: %v", err)
	}

	var fileContext string
	if fileID != nil {
		file, err := c.db.GetFileByID(*fileID)
		if err == nil && file != nil {
			fileContext = file.ProcessedData
		} else if err != nil {
			log.Printf("Failed to get file context for file ID %d: %v", *fileID, err)
		}
	}

	context := c.ai.BuildContext(history, fileContext)

	aiResponse, err := c.ai.GetChatResponse(userMessage, context)
	if err != nil {
		log.Printf("Failed to get AI response: %v", err)
		aiResponse = "I apologize, but I encountered an issue while processing your request with the AI. Please try again."
	} else {
		aiMsg := &models.ChatMessage{
			UserID:  c.userID,
			Role:    "assistant",
			Content: aiResponse,
		}
		savedAIMsg, saveErr := c.db.CreateChatMessage(aiMsg)
		if saveErr != nil {
			log.Printf("Failed to save AI message: %v", saveErr)
		} else {
			aiMsgResponse := Message{
				Type:    "chat",
				Content: aiResponse,
				Data: map[string]interface{}{
					"id":         savedAIMsg.ID,
					"role":       "assistant",
					"created_at": savedAIMsg.CreatedAt,
				},
			}
			aiMsgBytes, _ := json.Marshal(aiMsgResponse)
			stopTypingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": false}}
			stopTypingBytes, _ := json.Marshal(stopTypingMsg)
			c.send <- stopTypingBytes
			c.send <- aiMsgBytes
			return
		}
	}

	stopTypingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": false}}
	stopTypingBytes, _ := json.Marshal(stopTypingMsg)
	c.send <- stopTypingBytes

	aiMsgResponse := Message{
		Type:    "chat",
		Content: aiResponse,
		Data: map[string]interface{}{
			"id":         0,
			"role":       "assistant",
			"created_at": time.Now(),
		},
	}
	aiMsgBytes, _ := json.Marshal(aiMsgResponse)
	c.send <- aiMsgBytes
}
