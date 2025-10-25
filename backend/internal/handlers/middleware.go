
package handlers

import (
	"context"
	"net/http"
	"strings"

	"trading-app/internal/auth"
	"trading-app/internal/database"
	"trading-app/pkg/utils"
)

type Middleware struct {
	db *database.DB
}

func NewMiddleware(db *database.DB) *Middleware {
	return &Middleware{db: db}
}

// AuthMiddleware validates JWT token and adds user_id to context
func (m *Middleware) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(w, http.StatusUnauthorized, "No authorization header")
			return
		}

		// Remove "Bearer " prefix
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		// Validate token
		userID, err := auth.ValidateToken(token)
		if err != nil {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		// Check if session exists and is valid
		session, err := m.db.GetSessionByToken(token)
		if err != nil {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Database error")
			return
		}
		if session == nil {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Session expired or invalid")
			return
		}

		// Add user_id to context
		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// CORSMiddleware handles CORS
func (m *Middleware) CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
