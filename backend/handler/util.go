package handler

import (
	"database/sql"
	"encoding/json"
	"sage/model"
	"time"
)

// SubColumns is the canonical list of subscription columns for SELECT queries.
// Update this in ONE place when adding/removing fields.
const SubColumns = "id, name, category, status, price, original_price, discount_note, currency, cycle, payment_method, start_date, next_renewal, url, notes, auto_renew, remind_days, created_at, updated_at"

// ScanSub scans a row into a Subscription struct. Use with SubColumns.
func ScanSub(scanner interface{ Scan(...interface{}) error }) (model.Subscription, error) {
	var s model.Subscription
	err := scanner.Scan(&s.ID, &s.Name, &s.Category, &s.Status, &s.Price, &s.OriginalPrice, &s.DiscountNote, &s.Currency, &s.Cycle, &s.PaymentMethod, &s.StartDate, &s.NextRenewal, &s.URL, &s.Notes, &s.AutoRenew, &s.RemindDays, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

// ScanSubRows scans all rows into a slice of Subscriptions.
func ScanSubRows(rows *sql.Rows) ([]model.Subscription, error) {
	var subs []model.Subscription
	for rows.Next() {
		s, err := ScanSub(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func daysUntil(dateStr string) *int {
	if dateStr == "" {
		return nil
	}
	d, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil
	}
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
	days := int(d.Sub(now).Hours() / 24)
	return &days
}

func monthlyEquivalent(price float64, cycle string) float64 {
	switch cycle {
	case "yearly":
		return price / 12
	case "quarterly":
		return price / 3
	case "weekly":
		return price * 52 / 12
	case "lifetime":
		return 0
	default: // monthly
		return price
	}
}

func yearlyEquivalent(price float64, cycle string) float64 {
	switch cycle {
	case "yearly":
		return price
	case "quarterly":
		return price * 4
	case "weekly":
		return price * 52
	case "lifetime":
		return 0
	default: // monthly
		return price * 12
	}
}

func subtractMonths(n int) string {
	now := time.Now()
	t := time.Date(now.Year(), now.Month()-time.Month(n), 1, 0, 0, 0, 0, now.Location())
	return t.Format("2006-01")
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// advanceRenewal advances next_renewal past today based on cycle.
// Returns the new date string, or original if no advancement needed.
func advanceRenewal(nextRenewal, cycle string) string {
	if nextRenewal == "" {
		return ""
	}
	d, err := time.Parse("2006-01-02", nextRenewal)
	if err != nil {
		return nextRenewal
	}
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for !d.After(now) {
		switch cycle {
		case "weekly":
			d = d.AddDate(0, 0, 7)
		case "monthly":
			d = d.AddDate(0, 1, 0)
		case "quarterly":
			d = d.AddDate(0, 3, 0)
		case "yearly":
			d = d.AddDate(1, 0, 0)
		default:
			return nextRenewal // lifetime or unknown
		}
	}
	return d.Format("2006-01-02")
}

// isExpiredNonRenew returns true if an active subscription has auto_renew=false and next_renewal is in the past.
// These subscriptions should NOT be counted in spending stats.
func isExpiredNonRenew(s model.Subscription) bool {
	if boolVal(s.AutoRenew, true) || s.NextRenewal == "" || s.Status != "active" {
		return false
	}
	d := daysUntil(s.NextRenewal)
	return d != nil && *d < 0
}

// boolVal safely dereferences a *bool, returning defaultVal if nil.
func boolVal(b *bool, defaultVal bool) bool {
	if b == nil {
		return defaultVal
	}
	return *b
}


