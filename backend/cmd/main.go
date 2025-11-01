package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"strconv"
	"trading-app/internal/ai"
	"trading-app/internal/auth"
	"trading-app/internal/database"
	"trading-app/internal/email"
	"trading-app/internal/handlers"
	"trading-app/internal/openalgo"
	"trading-app/internal/websocket"
)

func main() {
	// Load environment variables
	dbPath := getEnv("DB_PATH", "/root/trading-app/backend/data/trading.db")
	uploadDir := getEnv("UPLOAD_DIR", "./data/uploads")
	port := getEnv("PORT", "8080")
	// FIX: Ensure these are declared correctly for use below
	openalgoURL := getEnv("OPENALGO_URL", "https://openalgo.mywire.org")
	openalgoAPIKey := getEnv("OPENALGO_API_KEY", "")
	geminiAPIKey := getEnv("GEMINI_API_KEY", "")

	// Email configuration
	smtpHost := getEnv("SMTP_HOST", "")
	smtpPortStr := getEnv("SMTP_PORT", "587")
	smtpUsername := getEnv("SMTP_USERNAME", "")
	smtpPassword := getEnv("SMTP_PASSWORD", "")
	emailSender := getEnv("EMAIL_SENDER", "")
	emailRecipient := getEnv("EMAIL_RECIPIENT", "")
	smtpPort, _ := strconv.Atoi(smtpPortStr)

	// Create data directories
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	// Initialize database
	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create default admin user if none exists
	defaultPassword, _ := auth.HashPassword("admin123")
	if err := db.Initialize("admin", defaultPassword); err != nil {
		log.Printf("Warning: Failed to initialize default user: %v", err)
	}

	// Cleanup expired sessions periodically
	go func() {
		for {
			if err := db.CleanupExpiredSessions(); err != nil {
				log.Printf("Failed to cleanup sessions: %v", err)
			}
			// Run every hour
			time.Sleep(1 * time.Hour)
		}
	}()

	// FIX 1: Initialize OpenAlgo client with URL and API Key
	openalgoClient := openalgo.NewOpenAlgoClient(openalgoURL, openalgoAPIKey)

	// Initialize Email service
	emailService := email.NewEmailService(smtpHost, smtpPort, smtpUsername, smtpPassword, emailSender)

	// Initialize AI client
	aiClient := ai.NewAIClient(geminiAPIKey)

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)
	middleware := handlers.NewMiddleware(db)
	chatHandler := handlers.NewChatHandler(db)
	fileHandler := handlers.NewFileHandler(db, uploadDir)
	strategyHandler := handlers.NewStrategyHandler(db)
	// FIX 3: Pass openalgoClient to TradeHandler
	tradeHandler := handlers.NewTradeHandler(db, openalgoClient) // <-- FIX IS HERE
	portfolioHandler := handlers.NewPortfolioHandler(db, openalgoClient)
	backtestHandler := handlers.NewBacktestHandler(db, openalgoClient)
	// FIX 2: Pass OpenAlgo config to WebSocketHandler
	wsHandler := handlers.NewWebSocketHandler(hub, db, aiClient, openalgoURL, openalgoAPIKey, emailService, emailRecipient)

	// Setup router
	r := mux.NewRouter()

	// FIX: Moved /signal under the /api/ prefix to ensure routing works correctly
	r.HandleFunc("/api/signal", tradeHandler.HandleSignal).Methods("GET")

	// Public routes
	r.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST")
	r.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST")

	// Protected routes
	r.HandleFunc("/api/auth/profile", middleware.AuthMiddleware(authHandler.GetProfile)).Methods("GET")
	r.HandleFunc("/api/auth/logout", middleware.AuthMiddleware(authHandler.Logout)).Methods("POST")

	// Chat routes
	r.HandleFunc("/api/chat/messages", middleware.AuthMiddleware(chatHandler.GetMessages)).Methods("GET")
	r.HandleFunc("/api/chat/send", middleware.AuthMiddleware(chatHandler.SendMessage)).Methods("POST")

	// File routes
	r.HandleFunc("/api/files/upload", middleware.AuthMiddleware(fileHandler.UploadFile)).Methods("POST")
	r.HandleFunc("/api/files", middleware.AuthMiddleware(fileHandler.GetFiles)).Methods("GET")
	r.HandleFunc("/api/files/get", middleware.AuthMiddleware(fileHandler.GetFile)).Methods("GET")

	// Strategy routes
	r.HandleFunc("/api/strategies", middleware.AuthMiddleware(strategyHandler.GetStrategies)).Methods("GET")
	r.HandleFunc("/api/strategies/get", middleware.AuthMiddleware(strategyHandler.GetStrategy)).Methods("GET")
	r.HandleFunc("/api/strategies/create", middleware.AuthMiddleware(strategyHandler.CreateStrategy)).Methods("POST")
	r.HandleFunc("/api/strategies/status", middleware.AuthMiddleware(strategyHandler.UpdateStrategyStatus)).Methods("PUT")
	r.HandleFunc("/api/strategies/backtest-results", middleware.AuthMiddleware(strategyHandler.GetBacktestResults)).Methods("GET")

	// Backtest routes
	r.HandleFunc("/api/backtest/run", middleware.AuthMiddleware(backtestHandler.RunBacktest)).Methods("POST")

	// Trade routes
	r.HandleFunc("/api/trades", middleware.AuthMiddleware(tradeHandler.GetTrades)).Methods("GET")

	// Portfolio routes
	r.HandleFunc("/api/portfolio", middleware.AuthMiddleware(portfolioHandler.GetPortfolio)).Methods("GET")
	r.HandleFunc("/api/portfolio/positions", middleware.AuthMiddleware(portfolioHandler.GetPositions)).Methods("GET")
	r.HandleFunc("/api/portfolio/holdings", middleware.AuthMiddleware(portfolioHandler.GetHoldings)).Methods("GET")
	r.HandleFunc("/api/portfolio/order", middleware.AuthMiddleware(portfolioHandler.PlaceOrder)).Methods("POST")
	r.HandleFunc("/api/portfolio/quote", middleware.AuthMiddleware(portfolioHandler.GetQuote)).Methods("GET")

	// WebSocket route
	r.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	// Start server
	addr := ":" + port
	log.Printf("Server starting on %s", addr)
	log.Printf("OpenAlgo URL: %s", openalgoURL)
	log.Printf("Database: %s", dbPath)
	log.Printf("Upload directory: %s", uploadDir)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
