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
	geminiModel = "gemini-1.5-flash"
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
	cs := model.StartChat()

	lines := strings.Split(contextStr, "\n")
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
