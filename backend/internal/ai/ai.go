
package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"trading-app/internal/models"
)

// AIClient handles AI chat interactions
type AIClient struct {
	apiKey     string
	apiURL     string
	client     *http.Client
}

// NewAIClient creates a new AI client
func NewAIClient(apiKey string) *AIClient {
	// Using Abacus.AI LLM API
	return &AIClient{
		apiKey: apiKey,
		apiURL: "https://routellm.abacus.ai/v1/chat/completions",
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ChatRequest represents a chat request to the AI API
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents a chat response from the AI API
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
}

// Choice represents a choice in the chat response
type Choice struct {
	Message Message `json:"message"`
}

// GetChatResponse gets a response from the AI
func (ai *AIClient) GetChatResponse(userMessage, context string) (string, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: ai.getSystemPrompt(),
		},
	}

	// Add context if available
	if context != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: "Context: " + context,
		})
	}

	// Add user message
	messages = append(messages, Message{
		Role:    "user",
		Content: userMessage,
	})

	request := ChatRequest{
		Model:    "gemini-2.5-pro",
		Messages: messages,
		Stream:   false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", ai.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ai.apiKey)

	resp, err := ai.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// BuildContext builds context from chat history and file data
func (ai *AIClient) BuildContext(history []*models.ChatMessage, fileContext string) string {
	var contextBuilder strings.Builder

	// Add recent chat history
	if len(history) > 0 {
		contextBuilder.WriteString("Recent conversation:\n")
		for _, msg := range history {
			contextBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
		contextBuilder.WriteString("\n")
	}

	// Add file context
	if fileContext != "" {
		contextBuilder.WriteString("File data:\n")
		contextBuilder.WriteString(fileContext)
		contextBuilder.WriteString("\n")
	}

	return contextBuilder.String()
}

// getSystemPrompt returns the system prompt for the AI
func (ai *AIClient) getSystemPrompt() string {
	return `You are an AI trading assistant integrated into a trading application. You help users with:

1. Analyzing trading strategies and Pine Scripts
2. Interpreting CSV trading data and calculating metrics
3. Analyzing charts and market data
4. Providing insights on trades and positions
5. Explaining trading concepts and strategies
6. Helping with backtesting and strategy optimization

You have access to:
- User's uploaded files (Pine Scripts, CSV data, images, PDFs)
- OpenAlgo trading integration for live trading
- Historical trade data and performance metrics

Be concise, helpful, and focus on actionable insights. When analyzing data, provide specific numbers and percentages. When discussing strategies, explain the logic clearly.

If a user asks to place a trade, provide a summary and ask for confirmation before executing.`
}

// AnalyzeStrategy analyzes a trading strategy
func (ai *AIClient) AnalyzeStrategy(strategyCode string) (string, error) {
	prompt := fmt.Sprintf("Analyze this trading strategy and provide insights:\n\n%s", strategyCode)
	return ai.GetChatResponse(prompt, "")
}

// AnalyzeTradeData analyzes trade data
func (ai *AIClient) AnalyzeTradeData(data string) (string, error) {
	prompt := fmt.Sprintf("Analyze this trading data and provide insights:\n\n%s", data)
	return ai.GetChatResponse(prompt, "")
}
