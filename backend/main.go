package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sage/database"
	"sage/handler"
	"sage/middleware"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

//go:embed static/*
var staticFiles embed.FS

func maskToken(token string) string {
	if len(token) <= 12 {
		return "****"
	}
	return token[:8] + "..." + token[len(token)-4:]
}

func main() {
	dbPath := os.Getenv("SAGE_DB_PATH")
	if dbPath == "" {
		dbPath = "../data/sage.db"
	}
	port := os.Getenv("SAGE_PORT")
	if port == "" {
		port = "8321"
	}

	// Init database
	db, err := database.Init(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// JWT Secret: env var > DB > auto-generate
	jwtSecret := os.Getenv("SAGE_JWT_SECRET")
	if jwtSecret == "" {
		// Try reading from DB
		db.QueryRow("SELECT value FROM settings WHERE key = 'jwt_secret'").Scan(&jwtSecret)
	}
	if jwtSecret == "" {
		// Auto-generate and persist
		b := make([]byte, 32)
		rand.Read(b)
		jwtSecret = hex.EncodeToString(b)
		db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('jwt_secret', ?)", jwtSecret)
		log.Println("🔐 Generated new JWT secret (stored in DB)")
	}

	// Ensure API token exists (for agent/skill access)
	var apiToken string
	err = db.QueryRow("SELECT value FROM settings WHERE key = 'api_token'").Scan(&apiToken)
	if err != nil {
		b := make([]byte, 24)
		rand.Read(b)
		apiToken = "sage_" + hex.EncodeToString(b)
		db.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES ('api_token', ?)", apiToken)
		log.Printf("🔑 Generated new API token: %s", maskToken(apiToken))
	} else {
		log.Printf("🔑 API token: %s", maskToken(apiToken))
	}

	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())

	// Handlers
	authHandler := handler.NewAuthHandler(db, jwtSecret)
	subHandler := handler.NewSubHandler(db)
	statsHandler := handler.NewStatsHandler(db)
	exportHandler := handler.NewExportHandler(db)
	settingsHandler := handler.NewSettingsHandler(db)
	agentHandler := handler.NewAgentHandler(db)

	// Rate limiter for auth endpoints: 5 attempts per minute per IP
	authLimiter := middleware.NewRateLimiter(5, 1*time.Minute)
	authRateLimit := middleware.RateLimitMiddleware(authLimiter)

	// Public routes
	e.GET("/api/auth/status", authHandler.Status)
	e.POST("/api/auth/setup", authHandler.Setup, authRateLimit)
	e.POST("/api/auth/login", authHandler.Login, authRateLimit)
	e.GET("/api/agent/skill.md", agentHandler.SkillMD)

	// Protected routes
	api := e.Group("/api", middleware.JWTMiddleware(jwtSecret))
	api.GET("/subs", subHandler.List)
	api.POST("/subs", subHandler.Create)
	api.GET("/subs/:id", subHandler.Get)
	api.PUT("/subs/:id", subHandler.Update)
	api.DELETE("/subs/:id", subHandler.Delete)

	api.GET("/stats/overview", statsHandler.Overview)
	api.GET("/stats/by-category", statsHandler.ByCategory)
	api.GET("/stats/monthly-trend", statsHandler.MonthlyTrend)

	api.GET("/export/excel", exportHandler.Excel)
	api.GET("/export/json", exportHandler.JSON)
	api.POST("/import/json", exportHandler.ImportJSON)

	api.GET("/settings", settingsHandler.Get)
	api.PUT("/settings", settingsHandler.Update)
	api.GET("/settings/exchange-rates", settingsHandler.GetExchangeRates)
	api.POST("/settings/regenerate-token", settingsHandler.RegenerateToken)

	api.GET("/agent/status", agentHandler.Status)

	// Agent API routes (token-based auth, no login required)
	// Pass db so middleware reads current token from DB (supports hot-reload after regeneration)
	agentAPI := e.Group("/api", middleware.TokenAuthMiddleware(db))
	agentAPI.GET("/agent/subs", subHandler.List)
	agentAPI.POST("/agent/subs", subHandler.Create)
	agentAPI.GET("/agent/subs/:id", subHandler.Get)
	agentAPI.PUT("/agent/subs/:id", subHandler.Update)
	agentAPI.PATCH("/agent/subs/:id", subHandler.Patch)
	agentAPI.DELETE("/agent/subs/:id", subHandler.Delete)
	agentAPI.GET("/agent/subs/duplicates", subHandler.Duplicates)
	agentAPI.GET("/agent/stats/overview", statsHandler.Overview)
	agentAPI.GET("/agent/stats/by-category", statsHandler.ByCategory)
	agentAPI.GET("/agent/stats/upcoming", statsHandler.Upcoming)
	agentAPI.GET("/agent/stats/trend", statsHandler.MonthlyTrend)
	agentAPI.GET("/agent/stats/summary", statsHandler.Summary)

	// Serve static frontend (embedded)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err == nil {
		e.GET("/", echo.StaticFileHandler("index.html", staticFS))
		staticHandler := http.FileServer(http.FS(staticFS))
		e.GET("/*", echo.WrapHandler(staticHandler))
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nShutting down SubSage...")
		e.Close()
	}()

	log.Printf("🌱 SubSage starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
