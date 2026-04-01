package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"sage/model"

	"github.com/labstack/echo/v4"
	"github.com/xuri/excelize/v2"
)

type ExportHandler struct {
	db *sql.DB
}

func NewExportHandler(db *sql.DB) *ExportHandler {
	return &ExportHandler{db: db}
}

func (h *ExportHandler) Excel(c echo.Context) error {
	f := excelize.NewFile()
	sheet := "Subscriptions"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"名称", "分类", "状态", "价格", "原价", "折扣说明", "币种", "计费周期", "支付方式", "开始日期", "下次续费", "网址", "备注", "提醒天数", "创建时间"}
	categories := map[string]string{
		"ai": "AI 服务", "saas": "SaaS", "media": "影音", "domain": "域名", "vps": "VPS",
		"app": "应用", "vpn": "VPN", "dev": "开发工具", "membership": "会员", "gaming": "游戏",
		"edu": "教育", "storage": "云存储", "other": "其他",
	}
	statuses := map[string]string{"active": "活跃", "paused": "暂停", "cancelled": "已取消"}
	cycles := map[string]string{"monthly": "月付", "yearly": "年付", "quarterly": "季付", "weekly": "周付", "lifetime": "终身"}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Style header
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#3D7C5F"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, "A1", "O1", style)

	rows, err := h.db.Query(
		"SELECT name, category, status, price, original_price, discount_note, currency, cycle, payment_method, start_date, next_renewal, url, notes, remind_days, created_at FROM subscriptions ORDER BY CASE status WHEN 'active' THEN 0 WHEN 'paused' THEN 1 WHEN 'cancelled' THEN 2 END",
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	rowNum := 2
	for rows.Next() {
		var s model.Subscription
		rows.Scan(&s.Name, &s.Category, &s.Status, &s.Price, &s.OriginalPrice, &s.DiscountNote, &s.Currency, &s.Cycle, &s.PaymentMethod, &s.StartDate, &s.NextRenewal, &s.URL, &s.Notes, &s.RemindDays, &s.CreatedAt)

		originalPriceStr := ""
		if s.OriginalPrice != nil {
			originalPriceStr = fmt.Sprintf("%.2f", *s.OriginalPrice)
		}

		vals := []interface{}{
			s.Name,
			categories[s.Category],
			statuses[s.Status],
			s.Price,
			originalPriceStr,
			s.DiscountNote,
			s.Currency,
			cycles[s.Cycle],
			s.PaymentMethod,
			s.StartDate,
			s.NextRenewal,
			s.URL,
			s.Notes,
			s.RemindDays,
			s.CreatedAt,
		}
		for i, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, rowNum)
			f.SetCellValue(sheet, cell, v)
		}
		rowNum++
	}

	// Auto-width columns
	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 18)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=sage_export.xlsx")
	return c.Blob(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func (h *ExportHandler) JSON(c echo.Context) error {
	rows, err := h.db.Query(
		"SELECT "+SubColumns+" FROM subscriptions ORDER BY CASE status WHEN 'active' THEN 0 WHEN 'paused' THEN 1 WHEN 'cancelled' THEN 2 END",
	)
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

	c.Response().Header().Set("Content-Disposition", "attachment; filename=sage_export.json")
	return c.JSON(http.StatusOK, subs)
}

func (h *ExportHandler) ImportJSON(c echo.Context) error {
	var imported []model.Subscription
	if err := c.Bind(&imported); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
	}

	added := 0
	updated := 0

	for _, s := range imported {
		if s.Name == "" {
			continue
		}

		var existingID string
		err := h.db.QueryRow("SELECT id FROM subscriptions WHERE name = ?", s.Name).Scan(&existingID)

		if err == sql.ErrNoRows {
			s.ID = generateID()
			h.db.Exec(
				`INSERT INTO subscriptions (id, name, category, status, price, original_price, discount_note, currency, cycle, payment_method, start_date, next_renewal, url, notes, remind_days)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				s.ID, s.Name, s.Category, s.Status, s.Price, s.OriginalPrice, s.DiscountNote, s.Currency, s.Cycle, s.PaymentMethod, s.StartDate, s.NextRenewal, s.URL, s.Notes, s.RemindDays,
			)
			added++
		} else if err == nil {
			h.db.Exec(
				`UPDATE subscriptions SET category=?, status=?, price=?, original_price=?, discount_note=?, currency=?, cycle=?, payment_method=?, start_date=?, next_renewal=?, url=?, notes=?, remind_days=?, updated_at=datetime('now')
				 WHERE id=?`,
				s.Category, s.Status, s.Price, s.OriginalPrice, s.DiscountNote, s.Currency, s.Cycle, s.PaymentMethod, s.StartDate, s.NextRenewal, s.URL, s.Notes, s.RemindDays, existingID,
			)
			updated++
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"added":   added,
		"updated": updated,
		"message": fmt.Sprintf("导入完成：新增 %d，更新 %d", added, updated),
	})
}
