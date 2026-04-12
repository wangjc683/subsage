package handler

import (
	"database/sql"
	"net/http"
	"sage/model"

	"github.com/labstack/echo/v4"
)

type SubHandler struct {
	db *sql.DB
}

func NewSubHandler(db *sql.DB) *SubHandler {
	return &SubHandler{db: db}
}

// lazyAdvance checks auto_renew=true subscriptions and advances expired next_renewal dates.
func (h *SubHandler) lazyAdvance(subs []model.Subscription) []model.Subscription {
	for i, s := range subs {
		if boolVal(s.AutoRenew, true) && s.NextRenewal != "" && s.Status == "active" {
			newDate := advanceRenewal(s.NextRenewal, s.Cycle)
			if newDate != s.NextRenewal {
				subs[i].NextRenewal = newDate
				go h.db.Exec("UPDATE subscriptions SET next_renewal=?, updated_at=datetime('now') WHERE id=?", newDate, s.ID)
			}
		}
	}
	return subs
}

func (h *SubHandler) List(c echo.Context) error {
	category := c.QueryParam("category")
	status := c.QueryParam("status")
	search := c.QueryParam("search")
	sortBy := c.QueryParam("sort")
	sortOrder := c.QueryParam("order")

	query := "SELECT " + SubColumns + " FROM subscriptions WHERE 1=1"
	args := []interface{}{}

	if search != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+search+"%")
	}
	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	// Determine sort direction
	dir := "ASC"
	if sortOrder == "desc" {
		dir = "DESC"
	}
	statusOrder := "CASE status WHEN 'active' THEN 0 WHEN 'paused' THEN 1 WHEN 'cancelled' THEN 2 END"

	// Default sort: active first, then by next_renewal
	switch sortBy {
	case "price":
		query += " ORDER BY " + statusOrder + ", price " + dir
	case "name":
		query += " ORDER BY " + statusOrder + ", name " + dir
	case "next_renewal":
		query += " ORDER BY " + statusOrder + ", CASE WHEN next_renewal = '' THEN 1 ELSE 0 END, next_renewal " + dir
	case "created":
		query += " ORDER BY " + statusOrder + ", created_at " + dir
	default:
		query += " ORDER BY " + statusOrder + ", CASE WHEN next_renewal = '' THEN 1 ELSE 0 END, next_renewal ASC"
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	subs, err := ScanSubRows(rows)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if subs == nil {
		subs = []model.Subscription{}
	}

	// Lazy advance auto_renew subscriptions
	subs = h.lazyAdvance(subs)

	return c.JSON(http.StatusOK, subs)
}

func (h *SubHandler) Get(c echo.Context) error {
	id := c.Param("id")
	s, err := ScanSub(h.db.QueryRow(
		"SELECT "+SubColumns+" FROM subscriptions WHERE id = ?", id,
	))
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Lazy advance if auto_renew
	if boolVal(s.AutoRenew, true) && s.NextRenewal != "" && s.Status == "active" {
		newDate := advanceRenewal(s.NextRenewal, s.Cycle)
		if newDate != s.NextRenewal {
			s.NextRenewal = newDate
			go h.db.Exec("UPDATE subscriptions SET next_renewal=?, updated_at=datetime('now') WHERE id=?", newDate, s.ID)
		}
	}

	return c.JSON(http.StatusOK, s)
}

func (h *SubHandler) Create(c echo.Context) error {
	var s model.Subscription
	if err := c.Bind(&s); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if s.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	// Default auto_renew to true if not specified
	if s.AutoRenew == nil {
		t := true
		s.AutoRenew = &t
	}

	s.ID = generateID()
	_, err := h.db.Exec(
		`INSERT INTO subscriptions (id, name, category, status, price, original_price, discount_note, currency, cycle, payment_method, start_date, next_renewal, url, notes, auto_renew, remind_days)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.Category, s.Status, s.Price, s.OriginalPrice, s.DiscountNote, s.Currency, s.Cycle, s.PaymentMethod, s.StartDate, s.NextRenewal, s.URL, s.Notes, boolVal(s.AutoRenew, true), s.RemindDays,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, s)
}

func (h *SubHandler) Update(c echo.Context) error {
	id := c.Param("id")

	var s model.Subscription
	if err := c.Bind(&s); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	_, err := h.db.Exec(
		`UPDATE subscriptions SET name=?, category=?, status=?, price=?, original_price=?, discount_note=?, currency=?, cycle=?, payment_method=?, start_date=?, next_renewal=?, url=?, notes=?, auto_renew=?, remind_days=?, updated_at=datetime('now')
		 WHERE id=?`,
		s.Name, s.Category, s.Status, s.Price, s.OriginalPrice, s.DiscountNote, s.Currency, s.Cycle, s.PaymentMethod, s.StartDate, s.NextRenewal, s.URL, s.Notes, boolVal(s.AutoRenew, true), s.RemindDays, id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	s.ID = id
	return c.JSON(http.StatusOK, s)
}

func (h *SubHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	_, err := h.db.Exec("DELETE FROM subscriptions WHERE id = ?", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "deleted"})
}

// Patch partially updates a subscription — only non-zero/non-empty fields are updated.
func (h *SubHandler) Patch(c echo.Context) error {
	id := c.Param("id")

	// First, load existing subscription
	existing, err := ScanSub(h.db.QueryRow(
		"SELECT "+SubColumns+" FROM subscriptions WHERE id = ?", id,
	))
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Parse partial update from request body
	var patch model.Subscription
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Merge: only overwrite if patch field is non-zero
	if patch.Name != "" {
		existing.Name = patch.Name
	}
	if patch.Category != "" {
		existing.Category = patch.Category
	}
	if patch.Status != "" {
		existing.Status = patch.Status
	}
	if patch.Price != 0 {
		existing.Price = patch.Price
	}
	if patch.OriginalPrice != nil {
		existing.OriginalPrice = patch.OriginalPrice
	}
	if patch.DiscountNote != "" {
		existing.DiscountNote = patch.DiscountNote
	}
	if patch.Currency != "" {
		existing.Currency = patch.Currency
	}
	if patch.Cycle != "" {
		existing.Cycle = patch.Cycle
	}
	if patch.PaymentMethod != "" {
		existing.PaymentMethod = patch.PaymentMethod
	}
	if patch.StartDate != "" {
		existing.StartDate = patch.StartDate
	}
	if patch.NextRenewal != "" {
		existing.NextRenewal = patch.NextRenewal
	}
	if patch.URL != "" {
		existing.URL = patch.URL
	}
	if patch.Notes != "" {
		existing.Notes = patch.Notes
	}
	if patch.AutoRenew != nil {
		existing.AutoRenew = patch.AutoRenew
	}
	if patch.RemindDays != 0 {
		existing.RemindDays = patch.RemindDays
	}

	_, err = h.db.Exec(
		`UPDATE subscriptions SET name=?, category=?, status=?, price=?, original_price=?, discount_note=?, currency=?, cycle=?, payment_method=?, start_date=?, next_renewal=?, url=?, notes=?, auto_renew=?, remind_days=?, updated_at=datetime('now')
		 WHERE id=?`,
		existing.Name, existing.Category, existing.Status, existing.Price, existing.OriginalPrice, existing.DiscountNote, existing.Currency, existing.Cycle, existing.PaymentMethod, existing.StartDate, existing.NextRenewal, existing.URL, existing.Notes, boolVal(existing.AutoRenew, true), existing.RemindDays, id,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	existing.ID = id
	return c.JSON(http.StatusOK, existing)
}

// Duplicates returns subscriptions with similar names to help avoid duplicate entries.
func (h *SubHandler) Duplicates(c echo.Context) error {
	name := c.QueryParam("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name parameter is required"})
	}

	// Search for similar names using LIKE
	query := "SELECT " + SubColumns + " FROM subscriptions WHERE name LIKE ? ORDER BY name ASC"
	rows, err := h.db.Query(query, "%"+name+"%")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	subs, err := ScanSubRows(rows)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if subs == nil {
		subs = []model.Subscription{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"query":      name,
		"duplicates": subs,
		"count":      len(subs),
	})
}
