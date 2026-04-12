package handler

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
)

type AgentHandler struct {
	db *sql.DB
}

func NewAgentHandler(db *sql.DB) *AgentHandler {
	return &AgentHandler{db: db}
}

type agentLogEntry struct {
	Method    string `json:"method"`
	Path      string `json:"path"`
	UserAgent string `json:"user_agent"`
	CreatedAt string `json:"created_at"`
}

func (h *AgentHandler) Status(c echo.Context) error {
	// Get masked API token
	var apiToken string
	h.db.QueryRow("SELECT value FROM settings WHERE key = 'api_token'").Scan(&apiToken)
	masked := maskAPIToken(apiToken)

	// Count today's calls
	var todayCount int
	h.db.QueryRow("SELECT COUNT(*) FROM agent_logs WHERE date(created_at) = date('now')").Scan(&todayCount)

	// Get last call time
	var lastCallAt sql.NullString
	h.db.QueryRow("SELECT created_at FROM agent_logs ORDER BY id DESC LIMIT 1").Scan(&lastCallAt)

	// Get recent calls (last 10)
	rows, err := h.db.Query("SELECT method, path, user_agent, created_at FROM agent_logs ORDER BY id DESC LIMIT 10")
	var recentCalls []agentLogEntry
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var entry agentLogEntry
			if err := rows.Scan(&entry.Method, &entry.Path, &entry.UserAgent, &entry.CreatedAt); err == nil {
				recentCalls = append(recentCalls, entry)
			}
		}
	}
	if recentCalls == nil {
		recentCalls = []agentLogEntry{}
	}

	// Get total call count
	var totalCount int
	h.db.QueryRow("SELECT COUNT(*) FROM agent_logs").Scan(&totalCount)

	// Only include full token if explicitly requested
	result := map[string]interface{}{
		"has_activity":      totalCount > 0,
		"total_calls":       totalCount,
		"total_calls_today": todayCount,
		"last_call_at":      lastCallAt.String,
		"recent_calls":      recentCalls,
		"api_token_masked":  masked,
	}
	if c.QueryParam("reveal") == "1" {
		result["api_token"] = apiToken
	}
	return c.JSON(http.StatusOK, result)
}

func maskAPIToken(token string) string {
	if len(token) <= 12 {
		return "****"
	}
	return token[:8] + "..." + token[len(token)-4:]
}

// SkillMD serves a dynamically generated SKILL.md for OpenClaw installation.
// This endpoint is public (no auth) so OpenClaw can fetch it directly.
func (h *AgentHandler) SkillMD(c echo.Context) error {
	// Read API token from DB
	var apiToken string
	h.db.QueryRow("SELECT value FROM settings WHERE key = 'api_token'").Scan(&apiToken)

	// Auto-detect base URL from request headers
	proto := c.Request().Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if c.Request().TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := c.Request().Header.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Request().Host
	}
	baseURL := proto + "://" + host

	skillContent := `---
name: subsage
description: Manage personal subscriptions (SaaS, AI, streaming, etc.) via SubSage REST API. Track spending, renewals, and categories.
---

# SubSage Subscription Manager

**SubSage** is a self-hosted subscription management tool. Use its REST API to manage subscriptions, view spending stats, and track renewals.

## Authentication

All requests require the ` + "`" + `X-API-Token` + "`" + ` header:
` + "```" + `
X-API-Token: ` + apiToken + `
` + "```" + `

Base URL: ` + baseURL + `/api/agent

## List Subscriptions

` + "```" + `bash
curl -s -H "X-API-Token: ` + apiToken + `" ` + baseURL + `/api/agent/subs
` + "```" + `

Filter by category, status, or search by name:
` + "```" + `bash
curl -s -H "X-API-Token: ` + apiToken + `" "` + baseURL + `/api/agent/subs?search=netflix"
curl -s -H "X-API-Token: ` + apiToken + `" "` + baseURL + `/api/agent/subs?category=ai&status=active"
` + "```" + `

## Create Subscription

` + "```" + `bash
curl -s -X POST -H "X-API-Token: ` + apiToken + `" -H "Content-Type: application/json" \
  -d '{"name":"ChatGPT Plus","price":20,"currency":"USD","cycle":"monthly","category":"ai"}' \
  ` + baseURL + `/api/agent/subs
` + "```" + `

Required fields: ` + "`" + `name` + "`" + `, ` + "`" + `price` + "`" + `
Optional fields: ` + "`" + `currency` + "`" + ` (default: CNY), ` + "`" + `cycle` + "`" + `, ` + "`" + `category` + "`" + `, ` + "`" + `status` + "`" + `, ` + "`" + `start_date` + "`" + `, ` + "`" + `next_renewal` + "`" + `, ` + "`" + `auto_renew` + "`" + ` (default: true), ` + "`" + `url` + "`" + `, ` + "`" + `notes` + "`" + `, ` + "`" + `payment_method` + "`" + `, ` + "`" + `original_price` + "`" + `, ` + "`" + `discount_note` + "`" + `

## Update Subscription (Full)

` + "```" + `bash
curl -s -X PUT -H "X-API-Token: ` + apiToken + `" -H "Content-Type: application/json" \
  -d '{"name":"ChatGPT Plus","price":25,"currency":"USD","cycle":"monthly","category":"ai","status":"active"}' \
  ` + baseURL + `/api/agent/subs/:id
` + "```" + `

## Partial Update (PATCH)

Only send the fields you want to change. Other fields are preserved:
` + "```" + `bash
curl -s -X PATCH -H "X-API-Token: ` + apiToken + `" -H "Content-Type: application/json" \
  -d '{"price":25}' \
  ` + baseURL + `/api/agent/subs/:id
` + "```" + `

## Delete Subscription

` + "```" + `bash
curl -s -X DELETE -H "X-API-Token: ` + apiToken + `" ` + baseURL + `/api/agent/subs/:id
` + "```" + `

## Check Duplicates

Before creating a new subscription, check if a similar one already exists:
` + "```" + `bash
curl -s -H "X-API-Token: ` + apiToken + `" "` + baseURL + `/api/agent/subs/duplicates?name=ChatGPT"
` + "```" + `

Returns ` + "`" + `{"query":"ChatGPT","duplicates":[...],"count":N}` + "`" + `

## Quick Summary

Get a concise overview of all subscriptions (useful for quick status checks):
` + "```" + `bash
curl -s -H "X-API-Token: ` + apiToken + `" ` + baseURL + `/api/agent/stats/summary
` + "```" + `

Returns: total count, active/paused/cancelled counts, monthly total, most expensive subscription, upcoming renewals count.

## Spending Overview

` + "```" + `bash
curl -s -H "X-API-Token: ` + apiToken + `" ` + baseURL + `/api/agent/stats/overview
` + "```" + `

Returns: monthly/yearly spend, active count, daily cost.

## Monthly Trend

Compare spending across months:
` + "```" + `bash
curl -s -H "X-API-Token: ` + apiToken + `" ` + baseURL + `/api/agent/stats/trend
` + "```" + `

Returns last 12 months of monthly spending data.

## Upcoming Renewals

` + "```" + `bash
curl -s -H "X-API-Token: ` + apiToken + `" "` + baseURL + `/api/agent/stats/upcoming?days=7"
` + "```" + `

## Category Stats

` + "```" + `bash
curl -s -H "X-API-Token: ` + apiToken + `" ` + baseURL + `/api/agent/stats/by-category
` + "```" + `

## Reference

Categories: ` + "`" + `ai` + "`" + `, ` + "`" + `video` + "`" + `, ` + "`" + `music` + "`" + `, ` + "`" + `software` + "`" + `, ` + "`" + `dev` + "`" + `, ` + "`" + `cloud` + "`" + `, ` + "`" + `security` + "`" + `, ` + "`" + `app` + "`" + `, ` + "`" + `gaming` + "`" + `, ` + "`" + `membership` + "`" + ` (custom categories also supported)
Cycles: ` + "`" + `monthly` + "`" + `, ` + "`" + `yearly` + "`" + `, ` + "`" + `quarterly` + "`" + `, ` + "`" + `weekly` + "`" + `, ` + "`" + `lifetime` + "`" + `
Status: ` + "`" + `active` + "`" + `, ` + "`" + `paused` + "`" + `, ` + "`" + `cancelled` + "`" + `
auto_renew: ` + "`" + `true` + "`" + ` (default, auto-renewal) or ` + "`" + `false` + "`" + ` (manual renewal, requires user action at expiry)
`

	c.Response().Header().Set("Content-Type", "text/markdown; charset=utf-8")
	return c.String(http.StatusOK, skillContent)
}
