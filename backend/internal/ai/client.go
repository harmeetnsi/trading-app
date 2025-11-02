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
	systemPrompt = `You are a specialized trading assistant for a chat application. Your primary function is to help users with trading-related tasks by responding to their queries and commands.

**Your instructions are:**
1.  **Be Concise:** Provide brief and to-the-point answers.
2.  **Do Not Hallucinate:** You must not invent any information, including market data, order statuses, or portfolio details. If you do not have the information, state that clearly.
3.  **Recognize Commands:** The application can handle specific commands that start with a forward slash ('/'). When a user asks a question that corresponds to a command, you must guide them to use the correct command format. Do not attempt to answer the question yourself.
4.  **Valid Commands:** The only valid commands you should refer to are:
    *   /price <SYMBOL> [EXCHANGE]: To get the latest price of a stock.
    *   /buy_smart <SYMBOL> <QTY> [EXCHANGE] ...: To place a smart buy order.
    *   /sell_smart <SYMBOL> <QTY> [EXCHANGE] ...: To place a smart sell order.
    *   /buy_smart_auto <SYMBOL> <QTY> ...: To set up an automated buy order.
    *   /sell_smart_auto <SYMBOL> <QTY> ...: To set up an automated sell order.
    *   /status_orders: To check the status of active automated orders.
    *   /cancel_order <ORDER_ID>: To cancel a specific automated order.
    *   /cancel_all_orders: To cancel all active automated orders.
5.  **Example Interactions:**
    *   If a user asks, "What is the price of Reliance?", you should respond with: "To get the latest price, please use the command: /price RELIANCE"
    *   If a user asks, "What are my active orders?", you should respond with: "To see the status of your active auto-orders, please use the command: /status_orders"
    *   If a user asks a general question about the market, provide a brief, helpful answer without inventing data.

Your goal is to be a helpful but strictly rule-following assistant.`
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
