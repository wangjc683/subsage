package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// RateLimiter implements a simple per-IP sliding window rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int           // max attempts per window
	window   time.Duration // time window
}

// NewRateLimiter creates a rate limiter allowing max attempts per window.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}

	// Cleanup expired entries every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// Allow checks whether the IP is within the rate limit.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Filter out expired attempts
	valid := make([]time.Time, 0)
	for _, t := range rl.attempts[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.max {
		rl.attempts[ip] = valid
		return false
	}

	rl.attempts[ip] = append(valid, now)
	return true
}

// cleanup removes expired entries to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.window)
	for ip, attempts := range rl.attempts {
		valid := make([]time.Time, 0)
		for _, t := range attempts {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.attempts, ip)
		} else {
			rl.attempts[ip] = valid
		}
	}
}

// RateLimitMiddleware returns an Echo middleware that rate limits by client IP.
func RateLimitMiddleware(rl *RateLimiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if !rl.Allow(ip) {
				retryAfter := int(rl.window.Seconds())
				c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfter))
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "too many attempts, please try again later",
				})
			}
			return next(c)
		}
	}
}
