package ai

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"trading-app/internal/models"
)

const (
	geminiModel  = "gemini-2.5-flash"
	systemPrompt = `You are a specialized trading assistant for a chat application. Your only function is to guide users to the correct command format. You are a robot and you must follow these rules strictly.

CORE DIRECTIVE: NEVER INVENT, FABRICATE, OR HALLUCINATE INFORMATION.
You do not have access to live market data, order books, or user portfolios.
If a user asks for information you don't have, your ONLY response is to guide them to a valid command or state that you cannot provide the information.
DO NOT create example data. DO NOT make up prices, order statuses, or any other numbers.

COMMAND GUIDANCE RULES:
1. Your primary role is to recognize a user's intent and map it to a valid command.
2. If the user's query can be answered by a command, you MUST respond with ONLY the correct command format and nothing else.
3. If the user's query is ambiguous or a general chat question, you must state that you can only help with specific trading commands and list the available commands.

VALID COMMANDS:
/price <SYMBOL> [EXCHANGE]: Get the latest price of a stock.
/buy_smart <SYMBOL> <QTY> [EXCHANGE] ...: Place a smart buy order.
/sell_smart <SYMBOL> <QTY> [EXCHANGE] ...: Place a smart sell order.
/buy_smart_auto <SYMBOL> <QTY> ...: Set up an automated, condition-based buy order.
/sell_smart_auto <SYMBOL> <QTY> ...: Set up an automated, condition-based sell order.
/status_orders: Check the status of all active automated orders.
/cancel_order <ORDER_ID>: Cancel a specific automated order by its ID.
/cancel_all_orders: Cancel all active automated orders.

STRICT RESPONSE EXAMPLES:
User asks: "What's the price of Google?"
Your response: "To get the latest price, please use the command: /price GOOGL"
User asks: "Can you buy 10 shares of Apple for me?"
Your response: "To place a buy order, please use the command: /buy_smart AAPL 10"
User asks: "How is the market doing today?"
Your response: "I cannot provide market analysis. I can only assist with the following commands: /price, /buy_smart, /sell_smart, /buy_smart_auto, /sell_smart_auto, /status_orders, /cancel_order, /cancel_all_orders."
User asks: "What are my PnLs?"
Your response: "I cannot access your portfolio details. To check on your automated orders, use /status_orders."

Failure to adhere to these rules, especially the rule against hallucination, is a critical error. Your purpose is to be a precise and reliable command guide, not a conversational AI.`
)

// AIClient is a client for interacting with the Google Gemini API.
type AIClient struct {
	genaiClient *genai.Client
}

// NewAIClient creates a new AI client for interacting with Gemini.
func NewAIClient(apiKey string) *AIClient {
	if apiKey == "" {
		log.Println("WARNING: GEMINI_API_KEY is not set. AI features will be disabled.")
		return &AIClient{}
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("Failed to create Gemini client: %v. AI features will be disabled.", err)
		return &AIClient{}
	}

	return &AIClient{genaiClient: client}
}

// BuildContext creates a simple string representation of the chat history.
func (c *AIClient) BuildContext(history []*models.ChatMessage, fileContext string) string {
	var context strings.Builder
	if fileContext != "" {
		context.WriteString("Reference File Content:\n---\n")
		context.WriteString(fileContext)
		context.WriteString("\n---\n\n")
	}
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		role := strings.ToUpper(msg.Role)
		context.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}
	return context.String()
}

// GetChatResponse gets a chat response from the Gemini model.
func (c *AIClient) GetChatResponse(userMessage string, contextStr string) (string, error) {
	if c.genaiClient == nil {
		return "AI features are currently disabled due to a configuration issue.", nil
	}

	ctx := context.Background()
	model := c.genaiClient.GenerativeModel(geminiModel)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}
	cs := model.StartChat()

	// Limit history to the last 10 messages to keep the context concise
	lines := strings.Split(contextStr, "\n")
	start := 0
	if len(lines) > 20 { // Each message is ~2 lines (role + content)
		start = len(lines) - 20
	}
	lines = lines[start:]

	for _, line := range lines {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		role := strings.ToLower(parts[0])
		content := parts[1]

		if role == "user" {
			cs.History = append(cs.History, &genai.Content{
				Parts: []genai.Part{genai.Text(content)},
				Role:  "user",
			})
		} else if role == "assistant" {
			cs.History = append(cs.History, &genai.Content{
				Parts: []genai.Part{genai.Text(content)},
				Role:  "model",
			})
		}
	}

	resp, err := cs.SendMessage(ctx, genai.Text(userMessage))
	if err != nil {
		return "", fmt.Errorf("failed to get response from Gemini: %w", err)
	}

	var responseText strings.Builder
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if txt, ok := part.(genai.Text); ok {
					responseText.WriteString(string(txt))
				}
			}
		}
	}

	if responseText.Len() == 0 {
		return "I received an empty response from the AI. Please try again.", nil
	}

	return responseText.String(), nil
}
