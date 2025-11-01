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
		Product:   product,
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
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered in sendError: %v", r)
		}
	}()
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

func (c *Client) monitorAndPlaceOrder(order *models.AutoOrder) {
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

	c.orderMux.Lock()
	cancelChan, ok := c.cancellation[order.ID]
	c.orderMux.Unlock()

	if !ok {
		log.Printf("AUTO-ORDER ERROR: Could not find cancellation channel for order %s. Stopping.", order.ID)
		return
	}

	checkDelay, _ := ParseIntervalDuration(order.Interval)
	if checkDelay < 5*time.Second {
		checkDelay = 5 * time.Second
	}
	ticker := time.NewTicker(checkDelay)
	defer ticker.Stop()

	expiryDuration := time.Until(order.ExpiresAt)
	if expiryDuration <= 0 {
		c.sendSystemMessage(fmt.Sprintf("⚠️ Auto-Order %s already expired. Stopping.", order.ID))
		return
	}
	if expiryDuration > 30*24*time.Hour {
		expiryDuration = 30 * 24 * time.Hour
	}
	expiryTimer := time.NewTimer(expiryDuration)
	defer expiryTimer.Stop()

	defer func() {
		c.removeAutoOrder(order.ID)
		log.Printf("AUTO-ORDER: Monitoring for %s (ID: %s) stopped and cleaned up.", order.Symbol, order.ID)
	}()

	for {
		select {
		case <-cancelChan:
			c.sendSystemMessage(fmt.Sprintf("❌ Auto-Order %s for %s was CANCELLED.", order.ID, order.Symbol))
			return
		case <-expiryTimer.C:
			c.sendSystemMessage(fmt.Sprintf("🕒 Auto-Order %s for %s has EXPIRED. Monitoring stopped.", order.ID, order.Symbol))
			return
		case <-ticker.C:
			if time.Now().After(order.ExpiresAt) {
				c.sendSystemMessage(fmt.Sprintf("🕒 Auto-Order %s for %s has EXPIRED. Monitoring stopped.", order.ID, order.Symbol))
				return
			}

			isMet, valuesMap, err := c.oaClient.EvaluatePineCondition(order.Interval, order.Condition, order.Symbol, order.Exchange)
			if err != nil {
				log.Printf("AUTO-ORDER: Evaluation error for %s: %v", order.ID, err)
				continue
			}

			if isMet {
				var indicatorSummary strings.Builder
				for name, value := range valuesMap {
					if math.IsNaN(value) || math.IsInf(value, 0) {
						indicatorSummary.WriteString(fmt.Sprintf(" **%s**: N/A |", name))
					} else {
						indicatorSummary.WriteString(fmt.Sprintf(" **%s**: %.2f |", name, value))
					}
				}

				orderReq := &openalgo.OpenAlgoSmartOrderRequest{
					Strategy:     "auto_chat",
					Symbol:       order.Symbol,
					Exchange:     order.Exchange,
					Action:       order.Action,
					Pricetype:    "MARKET",
					Product:      order.Product,
					Quantity:     order.Quantity,
				}

				log.Printf("AUTO-ORDER: Placing order for %s (ID: %s)", order.Symbol, order.ID)
				orderResponse, err := c.oaClient.PlaceOpenAlgoSmartOrder(orderReq)

				if err != nil {
					c.sendError(fmt.Sprintf("❌ Auto-Order %s FAILED to place order: %v. Monitoring continues.", order.ID, err))
					c.emailService.SendEmail(c.emailRecipient, "Auto-Order Execution Failed", fmt.Sprintf("Auto-Order %s failed to place order: %v", order.ID, err))
				} else {
					c.sendSystemMessage(fmt.Sprintf("✅ **AUTO ORDER EXECUTED** for %s on %s!\n\n### Trigger Values:\n%s\n**Order ID**: %s\n\nMonitoring continues.",
						order.Symbol, order.Exchange, indicatorSummary.String(), orderResponse.Data.OrderID))
					c.emailService.SendEmail(c.emailRecipient, "Auto-Order Executed", fmt.Sprintf("Auto-Order %s executed for %s on %s.", order.ID, order.Symbol, order.Exchange))
					go c.pollOrderStatus(order.ID, orderResponse.Data.OrderID)
				}
			}
		}
	}
}

func (c *Client) pollOrderStatus(autoOrderID, brokerOrderID string) {
	const maxRetries = 5
	const retryInterval = 15 * time.Second

	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryInterval)

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
			continue
		}

		log.Printf("Order %s status for %s (%s): %s", brokerOrderID, autoOrder.Symbol, autoOrder.Action, status.OrderStatus)

		switch strings.ToLower(status.OrderStatus) {
		case "complete":
			return
		case "rejected", "cancelled":
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
			return
		}
	}

	c.orderMux.Lock()
	autoOrder, exists := c.autoOrders[autoOrderID]
	c.orderMux.Unlock()
	if !exists {
		return
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

func (c *Client) removeAutoOrder(orderID string) {
	c.orderMux.Lock()
	order, exists := c.autoOrders[orderID]
	if !exists {
		c.orderMux.Unlock()
		return
	}

	order.CleanupOnce.Do(func() {
		log.Printf("AUTO-ORDER: Cleaning up order %s", orderID)
		delete(c.autoOrders, orderID)
		if ch, ok := c.cancellation[orderID]; ok {
			select {
			case <-ch:
			default:
				close(ch)
			}
			delete(c.cancellation, orderID)
		}
	})
	c.orderMux.Unlock()
}

func (c *Client) sendSystemMessage(content string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered in sendSystemMessage: %v", r)
		}
	}()
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
			// ... (existing implementation)
		case "/buy_smart", "/sell_smart":
			// ... (existing implementation)
		case "/rsi":
			// ... (existing implementation)
		case "/signal":
			// ... (existing implementation)
		case "/buy_smart_auto", "/sell_smart_auto":
			if len(parts) < 8 {
				responseContent = "Usage: `/buy_smart_auto <SYMBOL> <QTY> <EXCHANGE> <PRODUCT> <INTERVAL> <VALIDITY> <CONDITION...>`"
				break
			}
			action := "BUY"
			if cmd == "/sell_smart_auto" {
				action = "SELL"
			}
			symbol := strings.ToUpper(parts[1])
			quantityStr := parts[2]
			exchange := strings.ToUpper(parts[3])
			product := strings.ToUpper(parts[4])
			interval := strings.ToLower(parts[5])
			validityStr := strings.ToLower(parts[6])
			condition := strings.Join(parts[7:], " ")
			condition = strings.Trim(condition, "\"")
			if product != "MIS" && product != "NRML" && product != "CNC" {
				responseContent = fmt.Sprintf("Invalid product type: %s. Use MIS, NRML, or CNC.", product)
				break
			}
			quantity, err := strconv.Atoi(quantityStr)
			if err != nil || quantity <= 0 {
				responseContent = "Invalid quantity."
				break
			}
			if interval != "5m" && interval != "15m" && interval != "1h" {
				responseContent = fmt.Sprintf("Unsupported interval: %s.", interval)
				break
			}
			expiresAt, err := parseValidity(validityStr)
			if err != nil {
				responseContent = fmt.Sprintf("Invalid validity: %v.", err)
				break
			}
			_, initialValues, _ := c.oaClient.EvaluatePineCondition(interval, condition, symbol, exchange)
			var indicatorSummary strings.Builder
			for name, value := range initialValues {
				indicatorSummary.WriteString(fmt.Sprintf(" **%s**: %.2f |", name, value))
			}
			orderID, err := c.StartAutoOrderMonitoring(symbol, exchange, product, interval, condition, action, quantity, expiresAt)
			if err != nil {
				responseContent = fmt.Sprintf("❌ Failed to start auto order: %v", err)
			} else {
				expiryDisplay := "Running Indefinitely"
				if validityStr != "forever" {
					expiryDisplay = fmt.Sprintf("Expires at %s", expiresAt.Format("15:04:05 MST"))
				}
				responseContent = fmt.Sprintf("✅ **Auto Order Monitoring Started!**\n\n### Initial Values:\n%s\n- **ID**: %s\n- **Action**: %s\n- **Symbol**: %s on %s\n- **Interval**: %s\n- **Condition**: `%s`\n- **Validity**: %s",
					indicatorSummary.String(), orderID, action, symbol, exchange, interval, condition, expiryDisplay)
			}
		// ... (rest of the switch statement)
		}
	}

	assistMsg := &models.ChatMessage{
		UserID:  c.userID,
		Role:    "assistant",
		Content: responseContent,
	}
	savedAssistMsg, err := c.db.CreateChatMessage(assistMsg)
	if err != nil {
		log.Printf("Failed to save command response: %v", err)
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
		}
	}

	context := c.ai.BuildContext(history, fileContext)
	aiResponse, err := c.ai.GetChatResponse(userMessage, context)
	if err != nil {
		log.Printf("Failed to get AI response: %v", err)
		aiResponse = "I apologize, but I encountered an issue while processing your request with the AI. Please try again."
	}

	aiMsg := &models.ChatMessage{
		UserID:  c.userID,
		Role:    "assistant",
		Content: aiResponse,
	}
	savedAIMsg, saveErr := c.db.CreateChatMessage(aiMsg)
	if saveErr != nil {
		log.Printf("Failed to save AI message: %v", saveErr)
	}

	stopTypingMsg := Message{Type: "typing", Data: map[string]bool{"is_typing": false}}
	stopTypingBytes, _ := json.Marshal(stopTypingMsg)
	c.send <- stopTypingBytes

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
	c.send <- aiMsgBytes
}
