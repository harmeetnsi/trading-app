
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"trading-app/internal/auth"
	"trading-app/internal/database"
	"trading-app/internal/models"
	"trading-app/pkg/utils"
)

type AuthHandler struct {
	db *database.DB
}

func NewAuthHandler(db *database.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	if len(req.Password) < 6 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	// Check if user already exists
	existingUser, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}
	if existingUser != nil {
		utils.ErrorResponse(w, http.StatusConflict, "Username already exists")
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Create user
	user, err := h.db.CreateUser(req.Username, passwordHash)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	utils.SuccessResponse(w, "User registered successfully", user)
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get user
	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check password
	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		utils.ErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate token
	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Create session
	sessionID, err := auth.GenerateSessionID()
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	session := &models.Session{
		ID:        sessionID,
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(auth.TokenExpiry),
	}

	if err := h.db.CreateSession(session); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	response := LoginResponse{
		Token: token,
		User:  user,
	}

	utils.SuccessResponse(w, "Login successful", response)
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "No token provided")
		return
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	if err := h.db.DeleteSession(token); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to logout")
		return
	}

	utils.SuccessResponse(w, "Logout successful", nil)
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	user, err := h.db.GetUserByID(userID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		utils.ErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	utils.SuccessResponse(w, "Profile retrieved", user)
}
