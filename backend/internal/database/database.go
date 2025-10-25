
package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"trading-app/internal/models"
)

type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings for low memory
	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(2)
	conn.SetConnMaxLifetime(time.Hour)

	db := &DB{conn: conn}
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

// createTables creates all necessary database tables
func (db *DB) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		two_fa_enabled BOOLEAN DEFAULT 0,
		two_fa_secret TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		token TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS chat_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		file_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (file_id) REFERENCES files(id)
	);

	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		file_name TEXT NOT NULL,
		file_type TEXT NOT NULL,
		file_path TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		processed_data TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS strategies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		file_id INTEGER,
		code TEXT NOT NULL,
		status TEXT DEFAULT 'paused',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (file_id) REFERENCES files(id)
	);

	CREATE TABLE IF NOT EXISTS backtest_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		strategy_id INTEGER NOT NULL,
		start_date DATETIME NOT NULL,
		end_date DATETIME NOT NULL,
		initial_capital REAL NOT NULL,
		final_capital REAL NOT NULL,
		total_return REAL NOT NULL,
		total_trades INTEGER NOT NULL,
		winning_trades INTEGER NOT NULL,
		losing_trades INTEGER NOT NULL,
		max_drawdown REAL NOT NULL,
		sharpe_ratio REAL NOT NULL,
		result_data TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (strategy_id) REFERENCES strategies(id)
	);

	CREATE TABLE IF NOT EXISTS trades (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		strategy_id INTEGER,
		symbol TEXT NOT NULL,
		action TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		price REAL NOT NULL,
		order_type TEXT NOT NULL,
		status TEXT NOT NULL,
		order_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		executed_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (strategy_id) REFERENCES strategies(id)
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
	CREATE INDEX IF NOT EXISTS idx_chat_messages_user_id ON chat_messages(user_id);
	CREATE INDEX IF NOT EXISTS idx_files_user_id ON files(user_id);
	CREATE INDEX IF NOT EXISTS idx_strategies_user_id ON strategies(user_id);
	CREATE INDEX IF NOT EXISTS idx_trades_user_id ON trades(user_id);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// User operations
func (db *DB) CreateUser(username, passwordHash string) (*models.User, error) {
	result, err := db.conn.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, passwordHash,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return db.GetUserByID(int(id))
}

func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := db.conn.QueryRow(
		"SELECT id, username, password_hash, two_fa_enabled, two_fa_secret, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.TwoFAEnabled, &user.TwoFASecret, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (db *DB) GetUserByID(id int) (*models.User, error) {
	user := &models.User{}
	err := db.conn.QueryRow(
		"SELECT id, username, password_hash, two_fa_enabled, two_fa_secret, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.TwoFAEnabled, &user.TwoFASecret, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

// Session operations
func (db *DB) CreateSession(session *models.Session) error {
	_, err := db.conn.Exec(
		"INSERT INTO sessions (id, user_id, token, expires_at) VALUES (?, ?, ?, ?)",
		session.ID, session.UserID, session.Token, session.ExpiresAt,
	)
	return err
}

func (db *DB) GetSessionByToken(token string) (*models.Session, error) {
	session := &models.Session{}
	err := db.conn.QueryRow(
		"SELECT id, user_id, token, expires_at, created_at FROM sessions WHERE token = ? AND expires_at > datetime('now')",
		token,
	).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return session, err
}

func (db *DB) DeleteSession(token string) error {
	_, err := db.conn.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// Chat message operations
func (db *DB) CreateChatMessage(msg *models.ChatMessage) (*models.ChatMessage, error) {
	result, err := db.conn.Exec(
		"INSERT INTO chat_messages (user_id, role, content, file_id) VALUES (?, ?, ?, ?)",
		msg.UserID, msg.Role, msg.Content, msg.FileID,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return db.GetChatMessageByID(int(id))
}

func (db *DB) GetChatMessageByID(id int) (*models.ChatMessage, error) {
	msg := &models.ChatMessage{}
	err := db.conn.QueryRow(
		"SELECT id, user_id, role, content, file_id, created_at FROM chat_messages WHERE id = ?",
		id,
	).Scan(&msg.ID, &msg.UserID, &msg.Role, &msg.Content, &msg.FileID, &msg.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return msg, err
}

func (db *DB) GetChatMessagesByUserID(userID int, limit int) ([]*models.ChatMessage, error) {
	rows, err := db.conn.Query(
		"SELECT id, user_id, role, content, file_id, created_at FROM chat_messages WHERE user_id = ? ORDER BY created_at DESC LIMIT ?",
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*models.ChatMessage{}
	for rows.Next() {
		msg := &models.ChatMessage{}
		err := rows.Scan(&msg.ID, &msg.UserID, &msg.Role, &msg.Content, &msg.FileID, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// File operations
func (db *DB) CreateFile(file *models.File) (*models.File, error) {
	result, err := db.conn.Exec(
		"INSERT INTO files (user_id, file_name, file_type, file_path, file_size, processed_data) VALUES (?, ?, ?, ?, ?, ?)",
		file.UserID, file.FileName, file.FileType, file.FilePath, file.FileSize, file.ProcessedData,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return db.GetFileByID(int(id))
}

func (db *DB) GetFileByID(id int) (*models.File, error) {
	file := &models.File{}
	err := db.conn.QueryRow(
		"SELECT id, user_id, file_name, file_type, file_path, file_size, processed_data, created_at FROM files WHERE id = ?",
		id,
	).Scan(&file.ID, &file.UserID, &file.FileName, &file.FileType, &file.FilePath, &file.FileSize, &file.ProcessedData, &file.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return file, err
}

func (db *DB) GetFilesByUserID(userID int) ([]*models.File, error) {
	rows, err := db.conn.Query(
		"SELECT id, user_id, file_name, file_type, file_path, file_size, processed_data, created_at FROM files WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []*models.File{}
	for rows.Next() {
		file := &models.File{}
		err := rows.Scan(&file.ID, &file.UserID, &file.FileName, &file.FileType, &file.FilePath, &file.FileSize, &file.ProcessedData, &file.CreatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, nil
}

// Strategy operations
func (db *DB) CreateStrategy(strategy *models.Strategy) (*models.Strategy, error) {
	result, err := db.conn.Exec(
		"INSERT INTO strategies (user_id, name, description, file_id, code, status) VALUES (?, ?, ?, ?, ?, ?)",
		strategy.UserID, strategy.Name, strategy.Description, strategy.FileID, strategy.Code, strategy.Status,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return db.GetStrategyByID(int(id))
}

func (db *DB) GetStrategyByID(id int) (*models.Strategy, error) {
	strategy := &models.Strategy{}
	err := db.conn.QueryRow(
		"SELECT id, user_id, name, description, file_id, code, status, created_at, updated_at FROM strategies WHERE id = ?",
		id,
	).Scan(&strategy.ID, &strategy.UserID, &strategy.Name, &strategy.Description, &strategy.FileID, &strategy.Code, &strategy.Status, &strategy.CreatedAt, &strategy.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return strategy, err
}

func (db *DB) GetStrategiesByUserID(userID int) ([]*models.Strategy, error) {
	rows, err := db.conn.Query(
		"SELECT id, user_id, name, description, file_id, code, status, created_at, updated_at FROM strategies WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	strategies := []*models.Strategy{}
	for rows.Next() {
		strategy := &models.Strategy{}
		err := rows.Scan(&strategy.ID, &strategy.UserID, &strategy.Name, &strategy.Description, &strategy.FileID, &strategy.Code, &strategy.Status, &strategy.CreatedAt, &strategy.UpdatedAt)
		if err != nil {
			return nil, err
		}
		strategies = append(strategies, strategy)
	}

	return strategies, nil
}

func (db *DB) UpdateStrategyStatus(id int, status string) error {
	_, err := db.conn.Exec(
		"UPDATE strategies SET status = ?, updated_at = datetime('now') WHERE id = ?",
		status, id,
	)
	return err
}

// Trade operations
func (db *DB) CreateTrade(trade *models.Trade) (*models.Trade, error) {
	result, err := db.conn.Exec(
		"INSERT INTO trades (user_id, strategy_id, symbol, action, quantity, price, order_type, status, order_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		trade.UserID, trade.StrategyID, trade.Symbol, trade.Action, trade.Quantity, trade.Price, trade.OrderType, trade.Status, trade.OrderID,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return db.GetTradeByID(int(id))
}

func (db *DB) GetTradeByID(id int) (*models.Trade, error) {
	trade := &models.Trade{}
	err := db.conn.QueryRow(
		"SELECT id, user_id, strategy_id, symbol, action, quantity, price, order_type, status, order_id, created_at, executed_at FROM trades WHERE id = ?",
		id,
	).Scan(&trade.ID, &trade.UserID, &trade.StrategyID, &trade.Symbol, &trade.Action, &trade.Quantity, &trade.Price, &trade.OrderType, &trade.Status, &trade.OrderID, &trade.CreatedAt, &trade.ExecutedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return trade, err
}

func (db *DB) GetTradesByUserID(userID int, limit int) ([]*models.Trade, error) {
	rows, err := db.conn.Query(
		"SELECT id, user_id, strategy_id, symbol, action, quantity, price, order_type, status, order_id, created_at, executed_at FROM trades WHERE user_id = ? ORDER BY created_at DESC LIMIT ?",
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trades := []*models.Trade{}
	for rows.Next() {
		trade := &models.Trade{}
		err := rows.Scan(&trade.ID, &trade.UserID, &trade.StrategyID, &trade.Symbol, &trade.Action, &trade.Quantity, &trade.Price, &trade.OrderType, &trade.Status, &trade.OrderID, &trade.CreatedAt, &trade.ExecutedAt)
		if err != nil {
			return nil, err
		}
		trades = append(trades, trade)
	}

	return trades, nil
}

func (db *DB) UpdateTradeStatus(id int, status, orderID string) error {
	_, err := db.conn.Exec(
		"UPDATE trades SET status = ?, order_id = ?, executed_at = datetime('now') WHERE id = ?",
		status, orderID, id,
	)
	return err
}

// Backtest result operations
func (db *DB) CreateBacktestResult(result *models.BacktestResult) (*models.BacktestResult, error) {
	res, err := db.conn.Exec(
		"INSERT INTO backtest_results (strategy_id, start_date, end_date, initial_capital, final_capital, total_return, total_trades, winning_trades, losing_trades, max_drawdown, sharpe_ratio, result_data) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		result.StrategyID, result.StartDate, result.EndDate, result.InitialCapital, result.FinalCapital, result.TotalReturn, result.TotalTrades, result.WinningTrades, result.LosingTrades, result.MaxDrawdown, result.SharpeRatio, result.ResultData,
	)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return db.GetBacktestResultByID(int(id))
}

func (db *DB) GetBacktestResultByID(id int) (*models.BacktestResult, error) {
	result := &models.BacktestResult{}
	err := db.conn.QueryRow(
		"SELECT id, strategy_id, start_date, end_date, initial_capital, final_capital, total_return, total_trades, winning_trades, losing_trades, max_drawdown, sharpe_ratio, result_data, created_at FROM backtest_results WHERE id = ?",
		id,
	).Scan(&result.ID, &result.StrategyID, &result.StartDate, &result.EndDate, &result.InitialCapital, &result.FinalCapital, &result.TotalReturn, &result.TotalTrades, &result.WinningTrades, &result.LosingTrades, &result.MaxDrawdown, &result.SharpeRatio, &result.ResultData, &result.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return result, err
}

func (db *DB) GetBacktestResultsByStrategyID(strategyID int) ([]*models.BacktestResult, error) {
	rows, err := db.conn.Query(
		"SELECT id, strategy_id, start_date, end_date, initial_capital, final_capital, total_return, total_trades, winning_trades, losing_trades, max_drawdown, sharpe_ratio, result_data, created_at FROM backtest_results WHERE strategy_id = ? ORDER BY created_at DESC",
		strategyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []*models.BacktestResult{}
	for rows.Next() {
		result := &models.BacktestResult{}
		err := rows.Scan(&result.ID, &result.StrategyID, &result.StartDate, &result.EndDate, &result.InitialCapital, &result.FinalCapital, &result.TotalReturn, &result.TotalTrades, &result.WinningTrades, &result.LosingTrades, &result.MaxDrawdown, &result.SharpeRatio, &result.ResultData, &result.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Cleanup old sessions
func (db *DB) CleanupExpiredSessions() error {
	_, err := db.conn.Exec("DELETE FROM sessions WHERE expires_at < datetime('now')")
	return err
}

// Initialize creates a default admin user if no users exist
func (db *DB) Initialize(username, passwordHash string) error {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		log.Println("Creating default admin user...")
		_, err = db.CreateUser(username, passwordHash)
		return err
	}

	return nil
}
