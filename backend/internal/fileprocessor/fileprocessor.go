
package fileprocessor

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	//"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"
)

type FileProcessor struct{}

func NewFileProcessor() *FileProcessor {
	return &FileProcessor{}
}

// ProcessFile processes a file based on its type and returns JSON data
func (fp *FileProcessor) ProcessFile(filePath, fileType string) (string, error) {
	switch fileType {
	case "pine_script":
		return fp.processPineScript(filePath)
	case "csv":
		return fp.processCSV(filePath)
	case "image":
		return fp.processImage(filePath)
	case "pdf":
		return fp.processPDF(filePath)
	default:
		return "", fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// processPineScript extracts strategy information from Pine Script
func (fp *FileProcessor) processPineScript(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"type":    "pine_script",
		"content": string(content),
	}

	// Extract strategy name
	strategyNameRe := regexp.MustCompile(`strategy\s*\(\s*["']([^"']+)["']`)
	if matches := strategyNameRe.FindStringSubmatch(string(content)); len(matches) > 1 {
		data["strategy_name"] = matches[1]
	}

	// Extract indicator name
	indicatorNameRe := regexp.MustCompile(`indicator\s*\(\s*["']([^"']+)["']`)
	if matches := indicatorNameRe.FindStringSubmatch(string(content)); len(matches) > 1 {
		data["indicator_name"] = matches[1]
	}

	// Count entry/exit signals
	entryCount := strings.Count(string(content), "strategy.entry")
	exitCount := strings.Count(string(content), "strategy.exit") + strings.Count(string(content), "strategy.close")
	
	data["entry_signals"] = entryCount
	data["exit_signals"] = exitCount

	// Extract inputs
	inputRe := regexp.MustCompile(`input\s*\(\s*([^)]+)\s*\)`)
	inputs := []string{}
	for _, match := range inputRe.FindAllStringSubmatch(string(content), -1) {
		if len(match) > 1 {
			inputs = append(inputs, match[1])
		}
	}
	data["inputs"] = inputs

	jsonData, err := json.Marshal(data)
	return string(jsonData), err
}

// processCSV analyzes CSV trading data
func (fp *FileProcessor) processCSV(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	var records [][]string
	var err error

	if ext == ".xlsx" {
		records, err = fp.readExcel(filePath)
	} else {
		records, err = fp.readCSV(filePath)
	}

	if err != nil {
		return "", err
	}

	if len(records) < 2 {
		return "", fmt.Errorf("insufficient data in file")
	}

	data := map[string]interface{}{
		"type":       "csv_data",
		"rows":       len(records) - 1, // Exclude header
		"columns":    len(records[0]),
		"headers":    records[0],
		"preview":    records[1:min(6, len(records))], // First 5 rows
	}

	// Try to calculate basic trading metrics if it looks like trade data
	metrics := fp.calculateTradeMetrics(records)
	if metrics != nil {
		data["metrics"] = metrics
	}

	jsonData, err := json.Marshal(data)
	return string(jsonData), err
}

// readCSV reads a CSV file
func (fp *FileProcessor) readCSV(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	return reader.ReadAll()
}

// readExcel reads an Excel file
func (fp *FileProcessor) readExcel(filePath string) ([][]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Get the first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := f.GetRows(sheets[0])
	return rows, err
}

// calculateTradeMetrics attempts to calculate basic trade metrics
func (fp *FileProcessor) calculateTradeMetrics(records [][]string) map[string]interface{} {
	if len(records) < 2 {
		return nil
	}

	headers := records[0]
	
	// Try to find common column names
		pnlIdx := -1
		priceIdx := -1
		qtyIdx := -1
		_ = priceIdx
		_ = qtyIdx
	
	for i, header := range headers {
		lower := strings.ToLower(header)
		if strings.Contains(lower, "pnl") || strings.Contains(lower, "profit") || strings.Contains(lower, "loss") {
			pnlIdx = i
		}
		if strings.Contains(lower, "price") {
			priceIdx = i
		}
		if strings.Contains(lower, "qty") || strings.Contains(lower, "quantity") {
			qtyIdx = i
		}
	}

	metrics := map[string]interface{}{}
	
	if pnlIdx != -1 {
		totalPnL := 0.0
		winningTrades := 0
		losingTrades := 0
		
		for i := 1; i < len(records); i++ {
			if pnlIdx < len(records[i]) {
				pnl, err := strconv.ParseFloat(records[i][pnlIdx], 64)
				if err == nil {
					totalPnL += pnl
					if pnl > 0 {
						winningTrades++
					} else if pnl < 0 {
						losingTrades++
					}
				}
			}
		}
		
		metrics["total_pnl"] = totalPnL
		metrics["total_trades"] = len(records) - 1
		metrics["winning_trades"] = winningTrades
		metrics["losing_trades"] = losingTrades
		if winningTrades+losingTrades > 0 {
			metrics["win_rate"] = float64(winningTrades) / float64(winningTrades+losingTrades) * 100
		}
	}

	if len(metrics) == 0 {
		return nil
	}
	
	return metrics
}

// processImage processes image files (chart analysis placeholder)
func (fp *FileProcessor) processImage(filePath string) (string, error) {
	// For now, just return basic info
	// In production, you would integrate OCR or image analysis
	
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"type":     "image",
		"path":     filePath,
		"size":     fileInfo.Size(),
		"note":     "Image uploaded successfully. Chart analysis can be requested via AI chat.",
	}

	jsonData, err := json.Marshal(data)
	return string(jsonData), err
}

// processPDF extracts text from PDF files
func (fp *FileProcessor) processPDF(filePath string) (string, error) {
	file, r, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var textContent strings.Builder
	totalPages := r.NumPage()

	// Extract text from all pages (limit to first 10 pages for memory)
	maxPages := min(totalPages, 10)
	
	for pageNum := 1; pageNum <= maxPages; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		textContent.WriteString(text)
		textContent.WriteString("\n---\n")
	}

	data := map[string]interface{}{
		"type":        "pdf",
		"total_pages": totalPages,
		"extracted_pages": maxPages,
		"content":     textContent.String(),
	}

	jsonData, err := json.Marshal(data)
	return string(jsonData), err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
