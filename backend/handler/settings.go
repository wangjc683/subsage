package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type SettingsHandler struct {
	db *sql.DB
}

func NewSettingsHandler(db *sql.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

func (h *SettingsHandler) Get(c echo.Context) error {
	rows, err := h.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	settings := map[string]string{}
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		settings[k] = v
	}

	return c.JSON(http.StatusOK, settings)
}

func (h *SettingsHandler) Update(c echo.Context) error {
	var settings map[string]string
	if err := c.Bind(&settings); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	for k, v := range settings {
		h.db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", k, v)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "updated"})
}

func (h *SettingsHandler) GetExchangeRates(c echo.Context) error {
	var ratesJSON string
	var updatedAt string
	h.db.QueryRow("SELECT value FROM settings WHERE key = 'exchange_rates'").Scan(&ratesJSON)
	h.db.QueryRow("SELECT value FROM settings WHERE key = 'exchange_rates_updated'").Scan(&updatedAt)

	var rates interface{}
	json.Unmarshal([]byte(ratesJSON), &rates)

	// Check if we need to refresh (>24h old)
	if updatedAt == "" || isStale(updatedAt) {
		go RefreshExchangeRates(h.db)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"rates":     rates,
		"updated":   updatedAt,
		"base":      "USD",
	})
}

// RefreshExchangeRates fetches latest rates from open.er-api.com and stores them.
// This is a package-level function so both SettingsHandler and StatsHandler can use it.
func RefreshExchangeRates(db *sql.DB) {
	apiURL := "https://open.er-api.com/v6/latest/USD"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return
	}

	rates, ok := data["rates"].(map[string]interface{})
	if !ok {
		return
	}

	// Store as flat key-value: "USD_CNY" = 7.24
	flatRates := map[string]float64{}
	for cur, rate := range rates {
		if f, ok := rate.(float64); ok {
			flatRates["USD_"+cur] = f
		}
	}

	ratesJSON, _ := json.Marshal(flatRates)
	now := time.Now().Format("2006-01-02T15:04:05Z")

	db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('exchange_rates', ?)", string(ratesJSON))
	db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('exchange_rates_updated', ?)", now)
}

func isStale(updatedAt string) bool {
	t, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return true
	}
	return time.Since(t) > 24*time.Hour
}

// RegenerateToken generates a new API token, replacing the old one.
func (h *SettingsHandler) RegenerateToken(c echo.Context) error {
	b := make([]byte, 24)
	rand.Read(b)
	newToken := "sage_" + hex.EncodeToString(b)

	_, err := h.db.Exec("UPDATE settings SET value = ? WHERE key = 'api_token'", newToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to regenerate token"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"api_token": newToken,
		"message":   "Token regenerated. Update your agent configuration.",
	})
}
