package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"rbac/config"
	"rbac/utils"
)

func Temp2FAMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("X-2FA-Token")
		if raw == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing 2fa token",
			})
			c.Abort()
			return
		}

		claims, err := utils.Parse2FAToken(raw, cfg.JWT.AccessSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired 2fa session",
			})
			c.Abort()
			return
		}

		// ✅ PASS USER ID TO HANDLER
		c.Set("2fa_user_id", claims.UserID)
		c.Set("2fa_remember", claims.Remember)
		c.Next()
	}
}
