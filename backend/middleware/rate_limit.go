package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type clientRequest struct {
	count     int
	resetTime time.Time
}

func RateLimit(max int) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*clientRequest)

	// Clean up expired entries every minute
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			now := time.Now()
			for ip, req := range clients {
				if now.After(req.resetTime) {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		req, exists := clients[ip]

		// If client doesn't exist or reset time has passed, create new entry
		if !exists || now.After(req.resetTime) {
			clients[ip] = &clientRequest{
				count:     1,
				resetTime: now.Add(time.Minute),
			}
			c.Next()
			return
		}

		// Increment count
		req.count++

		// Check if exceeded limit
		if req.count > max {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"retry_after": int(req.resetTime.Sub(now).Seconds()),
			})
			return
		}

		c.Next()
	}
}
