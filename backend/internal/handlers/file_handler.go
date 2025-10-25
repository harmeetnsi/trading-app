
package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"trading-app/internal/database"
	"trading-app/internal/fileprocessor"
	"trading-app/internal/models"
	"trading-app/pkg/utils"
)

type FileHandler struct {
	db            *database.DB
	uploadDir     string
	fileProcessor *fileprocessor.FileProcessor
}

func NewFileHandler(db *database.DB, uploadDir string) *FileHandler {
	return &FileHandler{
		db:            db,
		uploadDir:     uploadDir,
		fileProcessor: fileprocessor.NewFileProcessor(),
	}
}

// UploadFile handles file uploads
func (h *FileHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	// Parse multipart form (32MB max)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	// Determine file type
	fileType := h.determineFileType(handler.Filename)
	if fileType == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Unsupported file type")
		return
	}

	// Create user directory if not exists
	userDir := filepath.Join(h.uploadDir, fmt.Sprintf("user_%d", userID))
	if err := os.MkdirAll(userDir, 0755); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create upload directory")
		return
	}

	// Generate unique filename
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%d_%s", timestamp, handler.Filename)
	filePath := filepath.Join(userDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to save file")
		return
	}
	defer dst.Close()

	fileSize, err := io.Copy(dst, file)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to write file")
		return
	}

	// Process file based on type
	processedData, err := h.fileProcessor.ProcessFile(filePath, fileType)
	if err != nil {
		// Log error but don't fail the upload
		processedData = fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}

	// Save file record to database
	fileRecord := &models.File{
		UserID:        userID,
		FileName:      handler.Filename,
		FileType:      fileType,
		FilePath:      filePath,
		FileSize:      fileSize,
		ProcessedData: processedData,
	}

	savedFile, err := h.db.CreateFile(fileRecord)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to save file record")
		return
	}

	utils.SuccessResponse(w, "File uploaded successfully", savedFile)
}

// GetFiles retrieves all files for the current user
func (h *FileHandler) GetFiles(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	files, err := h.db.GetFilesByUserID(userID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve files")
		return
	}

	utils.SuccessResponse(w, "Files retrieved", files)
}

// GetFile retrieves a specific file
func (h *FileHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	fileIDStr := r.URL.Query().Get("id")
	if fileIDStr == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "File ID is required")
		return
	}

	var fileID int
	if _, err := fmt.Sscanf(fileIDStr, "%d", &fileID); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid file ID")
		return
	}

	file, err := h.db.GetFileByID(fileID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve file")
		return
	}
	if file == nil {
		utils.ErrorResponse(w, http.StatusNotFound, "File not found")
		return
	}

	// Ensure user owns this file
	if file.UserID != userID {
		utils.ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	utils.SuccessResponse(w, "File retrieved", file)
}

// determineFileType determines the file type based on extension
func (h *FileHandler) determineFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	
	switch ext {
	case ".pine", ".txt":
		// Check if it's a Pine Script by content (simplified check)
		return "pine_script"
	case ".csv", ".xlsx":
		return "csv"
	case ".jpg", ".jpeg", ".png":
		return "image"
	case ".pdf":
		return "pdf"
	default:
		return ""
	}
}
