package auth

import (
	"github.com/gin-gonic/gin"

	"rbac/config"
	"rbac/handler"
	"rbac/middleware"
	"rbac/models"
)

// RegisterPublic mounts unauthenticated auth endpoints under /api/v1/auth.
func RegisterPublic(api *gin.RouterGroup, h *handler.AuthHandler, cfg *config.Config) {
	group := api.Group("/auth")
	group.POST("/login", middleware.RateLimit(10), middleware.BruteForceGuard(), h.Login)
	group.POST("/refresh", h.RefreshToken)
	group.POST("/forgot-password", h.ForgotPassword)
	group.POST("/reset-password", h.ResetPassword)
	group.POST("/verify-2fa", middleware.Temp2FAMiddleware(cfg), h.Verify2FA)
}

// RegisterProtected mounts authenticated profile / 2FA / logout routes.
func RegisterProtected(protected *gin.RouterGroup, h *handler.AuthHandler) {
	protected.POST("/logout", h.Logout)
	protected.GET("/profile", h.GetMe)
	protected.PUT("/profile", h.UpdateProfile)
	protected.POST("/change-password", h.ChangePassword)
	protected.POST("/2fa/enable", h.Enable2FA)
	protected.POST("/2fa/disable", h.Disable2FA)
}

// RegisterAdminUsers mounts admin user CRUD under /admin.
func RegisterAdminUsers(admin *gin.RouterGroup, h *handler.AuthHandler) {
	admin.POST("/users", h.CreateUser)
	admin.GET("/users", h.GetAllUsers)
	admin.PUT("/users/:id", h.EditUser)
	admin.DELETE("/users/:id", h.DeleteUser)
}

// RequireAdmin is exported for clarity at call sites.
func RequireAdmin() gin.HandlerFunc {
	return middleware.RequireRole(models.RoleAdmin)
}
