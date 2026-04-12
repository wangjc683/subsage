package handler

import (
	"database/sql"
	"net/http"
	"sage/model"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type StatsHandler struct {
	db *sql.DB
}

func NewStatsHandler(db *sql.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

func (h *StatsHandler) Overview(c echo.Context) error {
	// Get base currency
	baseCurrency := "CNY"
	h.db.QueryRow("SELECT value FROM settings WHERE key = 'base_currency'").Scan(&baseCurrency)

	// Active subscriptions stats
	active := []model.Subscription{}
	rows, err := h.db.Query(
		"SELECT "+SubColumns+" FROM subscriptions WHERE status = 'active'",
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	byCurrency := map[string]model.MonthlyInfo{}
	for rows.Next() {
		s, err := ScanSub(rows)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Lazy advance auto_renew subscriptions
		if boolVal(s.AutoRenew, true) && s.NextRenewal != "" {
			newDate := advanceRenewal(s.NextRenewal, s.Cycle)
			if newDate != s.NextRenewal {
				s.NextRenewal = newDate
				go h.db.Exec("UPDATE subscriptions SET next_renewal=?, updated_at=datetime('now') WHERE id=?", newDate, s.ID)
			}
		}

		active = append(active, s)

		// Skip expired non-auto-renew subscriptions from spending calculation
		if isExpiredNonRenew(s) {
			continue
		}

		m := monthlyEquivalent(s.Price, s.Cycle)
		y := yearlyEquivalent(s.Price, s.Cycle)

		info := byCurrency[s.Currency]
		info.Monthly += m
		info.Yearly += y
		byCurrency[s.Currency] = info
	}

	// Convert all to base currency
	rates := h.getRates()
	monthlyBase := 0.0
	yearlyBase := 0.0
	for cur, info := range byCurrency {
		rate := h.convertRate(cur, baseCurrency, rates)
		monthlyBase += info.Monthly * rate
		yearlyBase += info.Yearly * rate
	}

	// Upcoming (next 7 days) and overdue
	var upcomingCount, overdueCount int
	upcoming := []model.Subscription{}
	urows, _ := h.db.Query(
		"SELECT "+SubColumns+" FROM subscriptions WHERE status = 'active' AND next_renewal != '' ORDER BY next_renewal ASC",
	)
	if urows != nil {
		defer urows.Close()
		for urows.Next() {
			s, _ := ScanSub(urows)

			// Lazy advance auto_renew subscriptions
			if boolVal(s.AutoRenew, true) {
				newDate := advanceRenewal(s.NextRenewal, s.Cycle)
				if newDate != s.NextRenewal {
					s.NextRenewal = newDate
					go h.db.Exec("UPDATE subscriptions SET next_renewal=?, updated_at=datetime('now') WHERE id=?", newDate, s.ID)
				}
			}

			days := daysUntil(s.NextRenewal)
			if days != nil && *days >= 0 && *days <= 7 {
				upcomingCount++
				if len(upcoming) < 5 {
					upcoming = append(upcoming, s)
				}
			}
			// Only count overdue for non-auto-renew (auto_renew=true are already advanced)
			if days != nil && *days < 0 && !boolVal(s.AutoRenew, true) {
				overdueCount++
			}
		}
	}

	return c.JSON(http.StatusOK, model.OverviewStats{
		MonthlyTotal:  monthlyBase,
		YearlyTotal:   yearlyBase,
		ActiveCount:   len(active),
		UpcomingCount: upcomingCount,
		OverdueCount:  overdueCount,
		BaseCurrency:  baseCurrency,
		Upcoming:      upcoming,
		ByCurrency:    byCurrency,
	})
}

func (h *StatsHandler) ByCategory(c echo.Context) error {
	// Get base currency
	baseCurrency := "CNY"
	h.db.QueryRow("SELECT value FROM settings WHERE key = 'base_currency'").Scan(&baseCurrency)
	rates := h.getRates()

	// Query all subscriptions and calculate properly
	catRows, err := h.db.Query(
		"SELECT category, price, currency, cycle, status FROM subscriptions",
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer catRows.Close()

	// Need full subscription data for auto_renew filtering
	fullRows, err2 := h.db.Query(
		"SELECT "+SubColumns+" FROM subscriptions",
	)
	if err2 != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err2.Error()})
	}
	defer fullRows.Close()
	catRows.Close()

	catMap := map[string]*model.CategoryStats{}
	for fullRows.Next() {
		s, scanErr := ScanSub(fullRows)
		if scanErr != nil {
			continue
		}

		if _, ok := catMap[s.Category]; !ok {
			catMap[s.Category] = &model.CategoryStats{Category: s.Category}
		}

		catMap[s.Category].Count++
		if s.Status == "active" && !isExpiredNonRenew(s) {
			m := monthlyEquivalent(s.Price, s.Cycle)
			rate := h.convertRate(s.Currency, baseCurrency, rates)
			catMap[s.Category].MonthlyTotal += m * rate
			catMap[s.Category].YearlyTotal += m * rate * 12
		}
	}

	result := []model.CategoryStats{}
	for _, v := range catMap {
		result = append(result, *v)
	}

	return c.JSON(http.StatusOK, result)
}

func (h *StatsHandler) MonthlyTrend(c echo.Context) error {
	// Get base currency and exchange rates
	baseCurrency := "CNY"
	h.db.QueryRow("SELECT value FROM settings WHERE key = 'base_currency'").Scan(&baseCurrency)
	rates := h.getRates()

	// Load all subscriptions (not just active, for historical accuracy)
	rows, err := h.db.Query(
		"SELECT start_date, price, currency, cycle, status, updated_at FROM subscriptions",
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	type subInfo struct {
		startMonth string  // "2026-01" format
		price      float64 // monthly equivalent in base currency
		active     bool
	}

	var subs []subInfo
	for rows.Next() {
		var startDate, cur, cycle, status, updatedAt string
		var price float64
		rows.Scan(&startDate, &price, &cur, &cycle, &status, &updatedAt)

		m := monthlyEquivalent(price, cycle)
		rate := h.convertRate(cur, baseCurrency, rates)
		monthly := m * rate

		// Determine start month
		startMonth := ""
		if len(startDate) >= 7 {
			startMonth = startDate[:7] // "2026-01-15" -> "2026-01"
		}

		subs = append(subs, subInfo{
			startMonth: startMonth,
			price:      monthly,
			active:     status == "active",
		})
	}

	// Build trend for the past 12 months
	now := time.Now()
	currentMonth := now.Format("2006-01")
	trend := []model.MonthTrend{}

	for i := 11; i >= 0; i-- {
		month := subtractMonths(i)

		// Check if we have a stored snapshot for past months
		if month < currentMonth {
			var snapAmount float64
			err := h.db.QueryRow(
				"SELECT amount FROM monthly_snapshots WHERE month = ?", month,
			).Scan(&snapAmount)
			if err == nil {
				// Use stored snapshot
				trend = append(trend, model.MonthTrend{Month: month, Amount: snapAmount})
				continue
			}
		}

		// Calculate: sum monthly cost of subs that existed during this month
		total := 0.0
		for _, s := range subs {
			// A subscription contributes to a month if:
			// 1. It has no start_date (treat as always existed) OR start_month <= month
			// 2. For past months: currently active subs contribute to all months since start;
			//    cancelled/paused subs only contribute if still in "active" status
			//    (we don't have cancel_date, so cancelled subs don't contribute to past months)
			if !s.active && month < currentMonth {
				continue // Skip non-active subs for past months (conservative estimate)
			}

			if s.startMonth == "" || s.startMonth <= month {
				total += s.price
			}
		}

		trend = append(trend, model.MonthTrend{Month: month, Amount: total})

		// Auto-snapshot past months (not current month, it's still in progress)
		if month < currentMonth {
			h.db.Exec(
				"INSERT OR IGNORE INTO monthly_snapshots (month, amount, currency) VALUES (?, ?, ?)",
				month, total, baseCurrency,
			)
		}
	}

	return c.JSON(http.StatusOK, trend)
}

// Summary returns a concise text summary of subscription status for Agent use.
func (h *StatsHandler) Summary(c echo.Context) error {
	baseCurrency := "CNY"
	h.db.QueryRow("SELECT value FROM settings WHERE key = 'base_currency'").Scan(&baseCurrency)
	rates := h.getRates()

	// Count by status
	var activeCount, pausedCount, cancelledCount int
	h.db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE status = 'active'").Scan(&activeCount)
	h.db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE status = 'paused'").Scan(&pausedCount)
	h.db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE status = 'cancelled'").Scan(&cancelledCount)
	totalCount := activeCount + pausedCount + cancelledCount

	// Calculate monthly spend (excluding expired non-auto-renew)
	sumRows, err := h.db.Query(
		"SELECT "+SubColumns+" FROM subscriptions WHERE status = 'active'",
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer sumRows.Close()

	monthlyTotal := 0.0
	var topName string
	var topPrice float64
	for sumRows.Next() {
		s, scanErr := ScanSub(sumRows)
		if scanErr != nil {
			continue
		}
		if isExpiredNonRenew(s) {
			continue
		}
		m := monthlyEquivalent(s.Price, s.Cycle)
		rate := h.convertRate(s.Currency, baseCurrency, rates)
		monthlyTotal += m * rate
	}

	// Find most expensive subscription
	h.db.QueryRow(`SELECT name, price, currency, cycle FROM subscriptions
		WHERE status = 'active' ORDER BY
		CASE cycle
			WHEN 'monthly' THEN price
			WHEN 'yearly' THEN price/12
			WHEN 'quarterly' THEN price/3
			WHEN 'weekly' THEN price*52/12
			ELSE 0
		END DESC LIMIT 1`).Scan(&topName, &topPrice, new(string), new(string))

	// Upcoming in 7 days
	var upcomingCount int
	h.db.QueryRow(`SELECT COUNT(*) FROM subscriptions
		WHERE status = 'active' AND next_renewal != ''
		AND date(next_renewal) BETWEEN date('now') AND date('now', '+7 days')`).Scan(&upcomingCount)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total_subscriptions": totalCount,
		"active":              activeCount,
		"paused":              pausedCount,
		"cancelled":           cancelledCount,
		"monthly_total":       monthlyTotal,
		"base_currency":       baseCurrency,
		"most_expensive":      topName,
		"upcoming_7_days":     upcomingCount,
	})
}


func (h *StatsHandler) getRates() map[string]interface{} {
	var ratesJSON string
	h.db.QueryRow("SELECT value FROM settings WHERE key = 'exchange_rates'").Scan(&ratesJSON)
	var rates map[string]interface{}
	if err := jsonUnmarshal([]byte(ratesJSON), &rates); err != nil {
		return map[string]interface{}{}
	}

	// Lazy refresh: check if rates are stale (>24h) and trigger background update
	var updatedAt string
	h.db.QueryRow("SELECT value FROM settings WHERE key = 'exchange_rates_updated'").Scan(&updatedAt)
	if updatedAt == "" || isStale(updatedAt) {
		go RefreshExchangeRates(h.db)
	}

	return rates
}

func (h *StatsHandler) convertRate(from, to string, rates map[string]interface{}) float64 {
	if from == to {
		return 1.0
	}
	// Try direct rate
	if r, ok := rates[from+"_"+to]; ok {
		if f, ok := r.(float64); ok {
			return f
		}
	}
	// Try via USD
	fromUSD := 1.0
	toUSD := 1.0
	if from != "USD" {
		if r, ok := rates["USD_"+from]; ok {
			if f, ok := r.(float64); ok && f > 0 {
				fromUSD = 1.0 / f
			}
		}
	}
	if to != "USD" {
		if r, ok := rates["USD_"+to]; ok {
			if f, ok := r.(float64); ok {
				toUSD = f
			}
		}
	}
	return fromUSD * toUSD
}

// Upcoming returns active subscriptions with next_renewal in the next N days (default 7).
// Query params: ?days=14
func (h *StatsHandler) Upcoming(c echo.Context) error {
	days := 7
	if d := c.QueryParam("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			days = v
		}
	}

	rows, err := h.db.Query(
		"SELECT "+SubColumns+" FROM subscriptions WHERE status = 'active' AND next_renewal != '' ORDER BY next_renewal ASC",
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	type upcomingItem struct {
		model.Subscription
		DaysUntil int `json:"days_until"`
	}

	result := []upcomingItem{}
	for rows.Next() {
		s, _ := ScanSub(rows)
		// Lazy advance auto_renew subscriptions
		if boolVal(s.AutoRenew, true) {
			newDate := advanceRenewal(s.NextRenewal, s.Cycle)
			if newDate != s.NextRenewal {
				s.NextRenewal = newDate
				go h.db.Exec("UPDATE subscriptions SET next_renewal=?, updated_at=datetime('now') WHERE id=?", newDate, s.ID)
			}
		}
		d := daysUntil(s.NextRenewal)
		if d != nil && *d >= 0 && *d <= days {
			result = append(result, upcomingItem{Subscription: s, DaysUntil: *d})
		}
	}

	return c.JSON(http.StatusOK, result)
}
