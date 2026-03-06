package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func IPAllowList(allowed []string) gin.HandlerFunc {
	allowedMap := map[string]bool{}
	for _, ip := range allowed {
		allowedMap[ip] = true
	}

	return func(c *gin.Context) {
		if !allowedMap[c.ClientIP()] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "access_denied",
			})
			return
		}
		c.Next()
	}
}
