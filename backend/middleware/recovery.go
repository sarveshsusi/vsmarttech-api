package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SafeRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// ❌ NEVER expose panic details
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "internal_server_error",
		})
	})
}
