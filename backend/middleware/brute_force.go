package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type failedAttempt struct {
	count     int
	lastTry   time.Time
	lockedUntil *time.Time
}

var (
	bruteForceAttempts = make(map[string]*failedAttempt)
	bruteForceLocker   sync.Mutex
)

// BruteForceGuard tracks failed login attempts and locks accounts temporarily
func BruteForceGuard() gin.HandlerFunc {
	// Cleanup old entries every 10 minutes
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			bruteForceLocker.Lock()
			now := time.Now()
			for key, attempt := range bruteForceAttempts {
				// Remove entries older than 1 hour
				if now.Sub(attempt.lastTry) > time.Hour {
					delete(bruteForceAttempts, key)
				}
			}
			bruteForceLocker.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		bruteForceLocker.Lock()
		attempt, exists := bruteForceAttempts[ip]
		
		// Check if IP is currently locked
		if exists && attempt.lockedUntil != nil && time.Now().Before(*attempt.lockedUntil) {
			remaining := int(attempt.lockedUntil.Sub(time.Now()).Seconds())
			bruteForceLocker.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too_many_failed_attempts",
				"message": "Account temporarily locked due to multiple failed login attempts",
				"retry_after_seconds": remaining,
			})
			return
		}

		// If lock has expired, reset the attempt
		if exists && attempt.lockedUntil != nil && time.Now().After(*attempt.lockedUntil) {
			delete(bruteForceAttempts, ip)
		}
		
		bruteForceLocker.Unlock()

		// Continue to handler
		c.Next()

		// After handler completes, check if login failed
		status := c.Writer.Status()
		
		// Login failed (401 Unauthorized)
		if status == http.StatusUnauthorized {
			bruteForceLocker.Lock()
			defer bruteForceLocker.Unlock()

			attempt, exists := bruteForceAttempts[ip]
			if !exists {
				bruteForceAttempts[ip] = &failedAttempt{
					count:   1,
					lastTry: time.Now(),
				}
			} else {
				attempt.count++
				attempt.lastTry = time.Now()

				// Lock account after 5 failed attempts within 15 minutes
				if attempt.count >= 5 {
					lockUntil := time.Now().Add(15 * time.Minute)
					attempt.lockedUntil = &lockUntil
				}
			}
		} else if status == http.StatusOK {
			// Login successful - reset failed attempts for this IP
			bruteForceLocker.Lock()
			delete(bruteForceAttempts, ip)
			bruteForceLocker.Unlock()
		}
	}
}
