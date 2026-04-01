package model

import "time"

type Subscription struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	Status        string   `json:"status"`
	Price         float64  `json:"price"`
	OriginalPrice *float64 `json:"original_price,omitempty"`
	DiscountNote  string   `json:"discount_note,omitempty"`
	Currency      string   `json:"currency"`
	Cycle         string  `json:"cycle"`
	PaymentMethod string  `json:"payment_method"`
	StartDate     string  `json:"start_date"`
	NextRenewal   string  `json:"next_renewal"`
	URL           string  `json:"url"`
	Notes         string  `json:"notes"`
	RemindDays    int     `json:"remind_days"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type OverviewStats struct {
	MonthlyTotal  float64              `json:"monthly_total"`
	YearlyTotal   float64              `json:"yearly_total"`
	ActiveCount   int                  `json:"active_count"`
	UpcomingCount int                  `json:"upcoming_count"`
	OverdueCount  int                  `json:"overdue_count"`
	BaseCurrency  string               `json:"base_currency"`
	Upcoming      []Subscription       `json:"upcoming"`
	ByCurrency    map[string]MonthlyInfo `json:"by_currency"`
}

type MonthlyInfo struct {
	Monthly float64 `json:"monthly"`
	Yearly  float64 `json:"yearly"`
}

type CategoryStats struct {
	Category     string  `json:"category"`
	Count        int     `json:"count"`
	MonthlyTotal float64 `json:"monthly_total"`
	YearlyTotal  float64 `json:"yearly_total"`
}

type MonthTrend struct {
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
}
