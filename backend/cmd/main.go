package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
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

	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	defaultPassword, _ := auth.HashPassword("admin123")
	if err := db.Initialize("admin", defaultPassword); err != nil {
		log.Printf("Warning: Failed to initialize default user: %v", err)
	}

	go func() {
		for {
			if err := db.CleanupExpiredSessions(); err != nil {
				log.Printf("Failed to cleanup sessions: %v", err)
			}
			time.Sleep(1 * time.Hour)
		}
	}()

	openalgoClient := openalgo.NewOpenAlgoClient(openalgoURL, openalgoAPIKey)
	emailService := email.NewEmailService(smtpHost, smtpPort, smtpUsername, smtpPassword, emailSender)
	aiClient := ai.NewAIClient(geminiAPIKey)
	hub := websocket.NewHub()
	go hub.Run()

	authHandler := handlers.NewAuthHandler(db)
	middleware := handlers.NewMiddleware(db)
	chatHandler := handlers.NewChatHandler(db)
	fileHandler := handlers.NewFileHandler(db, uploadDir)
	strategyHandler := handlers.NewStrategyHandler(db)
	tradeHandler := handlers.NewTradeHandler(db, openalgoClient)
	portfolioHandler := handlers.NewPortfolioHandler(db, openalgoClient)
	backtestHandler := handlers.NewBacktestHandler(db, openalgoClient)
	wsHandler := handlers.NewWebSocketHandler(hub, db, aiClient, openalgoURL, openalgoAPIKey, emailService, emailRecipient)

	r := mux.NewRouter()
	r.HandleFunc("/api/signal", tradeHandler.HandleSignal).Methods("GET")
	r.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST")
	r.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/api/auth/profile", middleware.AuthMiddleware(authHandler.GetProfile)).Methods("GET")
	r.HandleFunc("/api/auth/logout", middleware.AuthMiddleware(authHandler.Logout)).Methods("POST")
	r.HandleFunc("/api/chat/messages", middleware.AuthMiddleware(chatHandler.GetMessages)).Methods("GET")
	r.HandleFunc("/api/chat/send", middleware.AuthMiddleware(chatHandler.SendMessage)).Methods("POST")
	r.HandleFunc("/api/files/upload", middleware.AuthMiddleware(fileHandler.UploadFile)).Methods("POST")
	r.HandleFunc("/api/files", middleware.AuthMiddleware(fileHandler.GetFiles)).Methods("GET")
	r.HandleFunc("/api/files/get", middleware.AuthMiddleware(fileHandler.GetFile)).Methods("GET")
	r.HandleFunc("/api/strategies", middleware.AuthMiddleware(strategyHandler.GetStrategies)).Methods("GET")
	r.HandleFunc("/api/strategies/get", middleware.AuthMiddleware(strategyHandler.GetStrategy)).Methods("GET")
	r.HandleFunc("/api/strategies/create", middleware.AuthMiddleware(strategyHandler.CreateStrategy)).Methods("POST")
	r.HandleFunc("/api/strategies/status", middleware.AuthMiddleware(strategyHandler.UpdateStrategyStatus)).Methods("PUT")
	r.HandleFunc("/api/strategies/backtest-results", middleware.AuthMiddleware(strategyHandler.GetBacktestResults)).Methods("GET")
	r.HandleFunc("/api/backtest/run", middleware.AuthMiddleware(backtestHandler.RunBacktest)).Methods("POST")
	r.HandleFunc("/api/trades", middleware.AuthMiddleware(tradeHandler.GetTrades)).Methods("GET")
	r.HandleFunc("/api/portfolio", middleware.AuthMiddleware(portfolioHandler.GetPortfolio)).Methods("GET")
	r.HandleFunc("/api/portfolio/positions", middleware.AuthMiddleware(portfolioHandler.GetPositions)).Methods("GET")
	r.HandleFunc("/api/portfolio/holdings", middleware.AuthMiddleware(portfolioHandler.GetHoldings)).Methods("GET")
	r.HandleFunc("/api/portfolio/order", middleware.AuthMiddleware(portfolioHandler.PlaceOrder)).Methods("POST")
	r.HandleFunc("/api/portfolio/quote", middleware.AuthMiddleware(portfolioHandler.GetQuote)).Methods("GET")
	r.HandleFunc("/ws", wsHandler.HandleWebSocket)
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	handler := c.Handler(r)

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
