package database

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Init(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'other',
			status TEXT NOT NULL DEFAULT 'active',
			price REAL NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'USD',
			cycle TEXT NOT NULL DEFAULT 'monthly',
			payment_method TEXT DEFAULT '',
			start_date TEXT DEFAULT '',
			next_renewal TEXT DEFAULT '',
			url TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			remind_days INTEGER NOT NULL DEFAULT 3,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS monthly_snapshots (
			month TEXT PRIMARY KEY,
			amount REAL NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'CNY',
			recorded_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS agent_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			user_agent TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('base_currency', 'CNY')`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('exchange_rates', '{"USD_CNY":7.24,"USD_EUR":0.92,"USD_GBP":0.79,"USD_JPY":149.5,"USD_HKD":7.82,"USD_TWD":32.4,"USD_KRW":1380,"USD_CAD":1.36,"USD_AUD":1.55,"USD_SGD":1.35,"USD_USD":1}')`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('exchange_rates_updated', '')`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return err
		}
	}

	// Safe column additions (idempotent)
	_ = db.QueryRow("SELECT original_price FROM subscriptions LIMIT 1")
	// If the column doesn't exist, the error will mention it
	addCols := []struct {
		col  string
		def  string
	}{
		{"original_price", "REAL"},
		{"discount_note", "TEXT DEFAULT ''"},
		{"auto_renew", "INTEGER NOT NULL DEFAULT 1"},
	}
	for _, ac := range addCols {
		var val interface{}
		err := db.QueryRow("SELECT " + ac.col + " FROM subscriptions LIMIT 1").Scan(&val)
		if err != nil {
			// Column likely doesn't exist, add it
			db.Exec("ALTER TABLE subscriptions ADD COLUMN " + ac.col + " " + ac.def)
		}
	}

	return nil
}
