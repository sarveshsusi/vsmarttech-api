package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	bruteForceMaxAttempts = 5
	bruteForceWindow      = 15 * time.Minute
	bruteForceLockFor     = 15 * time.Minute
)

type failedAttempt struct {
	tries       []time.Time
	lockedUntil *time.Time
}

var (
	bruteForceAttempts = make(map[string]*failedAttempt)
	bruteForceLocker   sync.Mutex
	bruteForceCleanup  sync.Once
)

func startBruteForceCleanup() {
	bruteForceCleanup.Do(func() {
		go func() {
			for {
				time.Sleep(10 * time.Minute)
				bruteForceLocker.Lock()
				now := time.Now()
				for key, attempt := range bruteForceAttempts {
					if attempt.lockedUntil != nil && now.Before(*attempt.lockedUntil) {
						continue
					}
					if len(attempt.tries) == 0 || now.Sub(attempt.tries[len(attempt.tries)-1]) > time.Hour {
						delete(bruteForceAttempts, key)
					}
				}
				bruteForceLocker.Unlock()
			}
		}()
	})
}

func pruneTries(tries []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-bruteForceWindow)
	kept := tries[:0]
	for _, t := range tries {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	return kept
}

func peekLoginEmail(c *gin.Context) string {
	body, err := c.GetRawData()
	if err != nil || len(body) == 0 {
		return ""
	}
	// Restore body so the login handler can still bind JSON.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var payload struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(payload.Email))
}

func bruteForceKey(c *gin.Context) string {
	ip := c.ClientIP()
	email := peekLoginEmail(c)
	if email == "" {
		return "ip:" + ip
	}
	return "ip:" + ip + "|email:" + email
}

// BruteForceGuard tracks failed login attempts and locks the IP+email pair temporarily.
func BruteForceGuard() gin.HandlerFunc {
	startBruteForceCleanup()

	return func(c *gin.Context) {
		key := bruteForceKey(c)
		now := time.Now()

		bruteForceLocker.Lock()
		attempt, exists := bruteForceAttempts[key]

		if exists && attempt.lockedUntil != nil && now.Before(*attempt.lockedUntil) {
			remaining := int(attempt.lockedUntil.Sub(now).Seconds())
			if remaining < 1 {
				remaining = 1
			}
			bruteForceLocker.Unlock()
			c.Header("Retry-After", fmt.Sprintf("%d", remaining))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":               "too_many_failed_attempts",
				"message":             "Account temporarily locked due to multiple failed login attempts.",
				"retry_after_seconds": remaining,
			})
			return
		}

		if exists && attempt.lockedUntil != nil && !now.Before(*attempt.lockedUntil) {
			delete(bruteForceAttempts, key)
		}
		bruteForceLocker.Unlock()

		c.Next()

		status := c.Writer.Status()
		if status == http.StatusUnauthorized {
			bruteForceLocker.Lock()
			defer bruteForceLocker.Unlock()

			now := time.Now()
			attempt, exists := bruteForceAttempts[key]
			if !exists {
				bruteForceAttempts[key] = &failedAttempt{
					tries: []time.Time{now},
				}
				return
			}

			attempt.tries = append(pruneTries(attempt.tries, now), now)
			attempt.lockedUntil = nil

			if len(attempt.tries) >= bruteForceMaxAttempts {
				lockUntil := now.Add(bruteForceLockFor)
				attempt.lockedUntil = &lockUntil
			}
			return
		}

		if status == http.StatusOK || status == http.StatusAccepted {
			bruteForceLocker.Lock()
			delete(bruteForceAttempts, key)
			bruteForceLocker.Unlock()
		}
	}
}
