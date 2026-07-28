package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateWindow struct {
	count     int
	expiresAt time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
	limit   int
	window  time.Duration
	now     func() time.Time
	hits    uint64
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		panic("rate limit must be positive")
	}
	if window <= 0 {
		panic("rate limit window must be positive")
	}
	return &RateLimiter{
		windows: make(map[string]rateWindow),
		limit:   limit,
		window:  window,
		now:     time.Now,
	}
}

func (limiter *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		now := limiter.now().UTC()
		key := c.ClientIP() + ":" + c.FullPath()
		allowed, remaining, resetAt := limiter.consume(key, now)

		c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
		if !allowed {
			retryAfter := max(1, int(resetAt.Sub(now).Seconds()+1))
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			abort(c, http.StatusTooManyRequests, "request.rate_limited", "请求过于频繁，请稍后重试。")
			return
		}
		c.Next()
	}
}

func (limiter *RateLimiter) consume(key string, now time.Time) (bool, int, time.Time) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	limiter.hits++
	if limiter.hits%256 == 0 {
		for candidate, window := range limiter.windows {
			if !now.Before(window.expiresAt) {
				delete(limiter.windows, candidate)
			}
		}
	}

	current, exists := limiter.windows[key]
	if !exists || !now.Before(current.expiresAt) {
		current = rateWindow{expiresAt: now.Add(limiter.window)}
	}
	if current.count >= limiter.limit {
		limiter.windows[key] = current
		return false, 0, current.expiresAt
	}
	current.count++
	limiter.windows[key] = current
	return true, limiter.limit - current.count, current.expiresAt
}
