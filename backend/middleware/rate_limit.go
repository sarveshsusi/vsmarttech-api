package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type clientRequest struct {
	count     int
	resetTime time.Time
}

// RateLimit caps requests per minute. Authenticated requests are keyed by
// user ID (set by AuthMiddleware) so office/NAT users do not share one IP bucket.
// Unauthenticated routes fall back to ClientIP.
func RateLimit(max int) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*clientRequest)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			now := time.Now()
			for key, req := range clients {
				if now.After(req.resetTime) {
					delete(clients, key)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		key := rateLimitKey(c)

		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		req, exists := clients[key]

		if !exists || now.After(req.resetTime) {
			clients[key] = &clientRequest{
				count:     1,
				resetTime: now.Add(time.Minute),
			}
			c.Next()
			return
		}

		req.count++

		if req.count > max {
			retryAfter := int(req.resetTime.Sub(now).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"retry_after": retryAfter,
			})
			return
		}

		c.Next()
	}
}

func rateLimitKey(c *gin.Context) string {
	if raw, ok := c.Get(CtxUserID); ok {
		switch v := raw.(type) {
		case string:
			if v != "" {
				return "user:" + v
			}
		case fmt.Stringer:
			s := v.String()
			if s != "" {
				return "user:" + s
			}
		}
	}
	return "ip:" + c.ClientIP()
}
