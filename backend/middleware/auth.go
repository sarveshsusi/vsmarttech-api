package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/config"
	"rbac/models"
	"rbac/utils"
)

/*
=====================
 Context Keys
=====================
*/
const (
	CtxUserID    = "user_id"
	CtxUserEmail = "user_email"
	CtxUserRole  = "user_role"
)

/*
=====================
 Auth Middleware
=====================
 Validates JWT access token
*/
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header missing",
			})
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization format",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := utils.ValidateToken(token, cfg.JWT.AccessSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		// 🔒 Parse UUID from JWT (CRITICAL)
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user identity",
			})
			return
		}

		// ✅ Set typed values into context
		c.Set(CtxUserID, userID)            // uuid.UUID
		c.Set(CtxUserEmail, claims.Email)   // string
		c.Set(CtxUserRole, claims.Role)     // models.Role

		c.Next()
	}
}

/*
=====================
 Role-Based Access
=====================
*/
func RequireRole(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get(CtxUserRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "user role missing",
			})
			return
		}

		userRole, ok := roleValue.(models.Role)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user role",
			})
			return
		}

		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "insufficient permissions",
		})
	}
}

/*
=====================
 Admin Shortcut
=====================
*/
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin)
}
