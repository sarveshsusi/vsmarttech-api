package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type attempt struct {
	count int
	last  time.Time
}

func BruteForceGuard() gin.HandlerFunc {
	var mu sync.Mutex
	attempts := make(map[string]*attempt)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		a, exists := attempts[ip]
		if !exists {
			a = &attempt{}
			attempts[ip] = a
		}

		if a.count >= 10 && time.Since(a.last) < 5*time.Minute {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "account_temporarily_locked",
			})
			return
		}

		a.count++
		a.last = time.Now()
		mu.Unlock()

		c.Next()
	}
}
