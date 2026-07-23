package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/config"
	"rbac/models"
	"rbac/service"
	"rbac/utils"
)

type AuthHandler struct {
	service *service.AuthService
	cfg     *config.Config
}

func NewAuthHandler(service *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		service: service,
		cfg:     cfg,
	}
}

/* =====================
   DTOs
===================== */

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}
type CreateUserRequest struct {
	Name  string      `json:"name" binding:"required"`
	Email string      `json:"email" binding:"required,email"`
	Role  models.Role `json:"role" binding:"required"`

	// Support Engineer fields
	Designation string `json:"designation"`
	Phone       string `json:"phone"`

	// Customer fields
	CompanyID     uuid.UUID `json:"company_id"`
	CompanyName   string    `json:"company_name"`
	ContactPerson string    `json:"contact_person"`
	Location      string    `json:"location"`
	Plant         string    `json:"plant"`
	Address       string    `json:"address"`
}

type Verify2FARequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Code   string    `json:"code" binding:"required,len=6"`
}

type LoginRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	RememberDevice bool   `json:"rememberDevice"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type UpdateProfileRequest struct {
	Name string `json:"name" binding:"required,min=2,max=120"`
}

/* =====================
   Cookie Helpers
===================== */

func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string) {
	secure := h.cfg.Server.Env == "production"

	// SameSite=Strict blocks cross-site cookie sends (CSRF on /auth/refresh)
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		token,
		int(h.cfg.JWT.RefreshExpiry.Seconds()),
		"/",
		"",
		secure,
		true, // HttpOnly
	)
}

func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	secure := h.cfg.Server.Env == "production"

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/",
		"",
		secure,
		true,
	)
}

/* =====================
   Handlers
===================== */

// Login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	resp, err := h.service.Login(
		c,
		req.Email,
		req.Password,
		req.RememberDevice, // 👈 checkbox
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)
	// 🔐 2FA required
	var twoFA *service.ErrTwoFARequired
	if err != nil && errors.As(err, &twoFA) {
		c.JSON(http.StatusOK, gin.H{
			"two_fa_required": true,
			"two_fa_token":    twoFA.TempToken,
		})
		return
	}

	// 🔒 Password reset required
	if err != nil && errors.Is(err, service.ErrPasswordResetRequired) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "password_reset_required",
		})
		return
	}

	// ❌ Invalid credentials
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid credentials",
		})
		return
	}

	// ✅ Normal login success
	h.setRefreshCookie(c, resp.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.AccessToken,
		"user":         resp.User,
	})
}

// Refresh token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		var req RefreshRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
			return
		}
		refreshToken = req.RefreshToken
	}

	resp, err := h.service.RefreshAccessToken(
		refreshToken,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}

	h.setRefreshCookie(c, resp.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.AccessToken,
		"user":         resp.User,
	})
}

// Logout
func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err == nil {
		userID := c.MustGet("user_id").(uuid.UUID)
		role := c.MustGet("user_role").(models.Role)

		_ = h.service.Logout(
			refreshToken,
			userID,
			role,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
		)
	}

	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// Admin: Create User
func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	createdBy := c.MustGet("user_id").(uuid.UUID)

	user, err := h.service.CreateUser(
		req.Name,
		req.Email,
		req.Role,
		createdBy,
		c.ClientIP(),
		c.GetHeader("User-Agent"),

		req.CompanyID,
		req.CompanyName,
		req.ContactPerson,
		req.Location,
		req.Plant,
		req.Address,
		req.Designation,
		req.Phone,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"message": "User created. Password setup email sent.",
	})
}

// Change password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(models.Role)

	err := h.service.ChangePassword(
		userID,
		req.OldPassword,
		req.NewPassword,
		role,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "password changed, login again"})
}

// Update own profile (name)
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required (2–120 characters)"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	user, err := h.service.UpdateProfile(userID, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	})
}

// Get current user
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	user, err := h.service.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":    user.ID,
		"name":  user.Name, // ✅ FROM DB
		"email": user.Email,
		"role":  user.Role,
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.service.ResetPassword(req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "password reset successful",
	})
}
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 🔒 Do NOT reveal if user exists
	_ = h.service.SendPasswordResetEmail(req.Email)

	c.JSON(http.StatusOK, gin.H{
		"message": "If an account exists, a reset link has been sent.",
	})
}

// handler/auth_handler.go

type VerifyOTPRequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

// func (h *AuthHandler) Verify2FA(c *gin.Context) {
// 	userID := c.MustGet("temp_user_id").(uuid.UUID)

// 	var req VerifyOTPRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(400, gin.H{"error": "invalid request"})
// 		return
// 	}

// 	resp, err := h.service.Verify2FA(userID, req.Code)
// 	if err != nil {
// 		c.JSON(401, gin.H{"error": "invalid request"})
// 		return
// 	}

// 	h.setRefreshCookie(c, resp.RefreshToken)

// 	c.JSON(200, gin.H{
// 		"access_token": resp.AccessToken,
// 		"user":         resp.User,
// 	})
// }

// func (h *AuthHandler) Verify2FA(c *gin.Context) {
// 	var req Verify2FARequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
// 		return
// 	}

// 	resp, err := h.service.Verify2FA(req.UserID, req.Code)
// 	if err != nil {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"error": err.Error(),
// 		})
// 		return
// 	}

// 	h.setRefreshCookie(c, resp.RefreshToken)

//		c.JSON(http.StatusOK, gin.H{
//			"access_token": resp.AccessToken,
//			"user":         resp.User,
//		})
//	}
func (h *AuthHandler) Enable2FA(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.service.Enable2FA(userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to enable 2FA",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "2FA enabled successfully",
	})
}
func (h *AuthHandler) Disable2FA(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.service.Disable2FA(userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to disable 2FA",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "2FA disabled successfully",
	})
}

// func (h *AuthHandler) Verify2FA(c *gin.Context) {

// 	rawToken := c.GetHeader("X-2FA-Token")
// 	if rawToken == "" {
// 		c.JSON(401, gin.H{"error": "missing 2fa token"})
// 		return
// 	}

// 	claims, err := utils.Parse2FAToken(
// 		rawToken,
// 		h.cfg.JWT.AccessSecret,
// 	)
// 	if err != nil {
// 		c.JSON(401, gin.H{"error": "invalid 2fa session"})
// 		return
// 	}

// 	var req VerifyOTPRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(400, gin.H{"error": "invalid request"})
// 		return
// 	}

// 	resp, err := h.service.Verify2FA(
// 		claims.UserID,
// 		req.Code,
// 	)
// 	if err != nil {
// 		c.JSON(401, gin.H{"error": "invalid request"})
// 		return
// 	}

// 	h.setRefreshCookie(c, resp.RefreshToken)

// 	c.JSON(200, gin.H{
// 		"access_token": resp.AccessToken,
// 		"user": resp.User,
// 	})
// }

func (h *AuthHandler) Verify2FA(c *gin.Context) {
	userID := c.MustGet("2fa_user_id").(uuid.UUID)
	remember := c.MustGet("2fa_remember").(bool)

	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	resp, rememberToken, err := h.service.Verify2FA(
		userID,
		req.Code,
		remember,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid request",
		})
		return
	}

	// 🍪 Set remember-device cookie (ONLY if requested)
	if rememberToken != nil {
		c.SetCookie(
			"remember_device",
			*rememberToken,
			30*24*3600,
			"/",
			"",
			true, // Secure
			true, // HttpOnly
		)
		c.Writer.Header().Add(
			"Set-Cookie",
			"remember_device="+*rememberToken+"; SameSite=Strict",
		)
	}

	h.setRefreshCookie(c, resp.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.AccessToken,
		"user":         resp.User,
	})
}

// Admin: Get Users (Paginated)
// Admin: Get Users (Paginated)
func (h *AuthHandler) GetAllUsers(c *gin.Context) {
	role := c.Query("role")
	if role != "" {
		// If role is provided (e.g. for dropdowns), return all users of that role (no pagination for now)
		users, err := h.service.GetUsersByRole(models.Role(role))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
			return
		}
		c.JSON(http.StatusOK, users) // Returns ARRAY directly
		return
	}

	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			page = parsed
		}
	}

	users, total, err := h.service.GetUsersPaginated(page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"page":      page,
		"page_size": 3,
		"total":     total,
		"users":     users,
	})
}

func (h *AuthHandler) GetSupportEngineers(c *gin.Context) {
	users, err := h.service.GetUsersByRole(models.RoleSupport)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch support engineers"})
		return
	}
	c.JSON(http.StatusOK, users)
}

// ========================
// EDIT USER
// ========================
type EditUserRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Role     string `json:"role" binding:"required"`
	IsActive bool   `json:"is_active"`
}

func (h *AuthHandler) EditUser(c *gin.Context) {
	userID := c.Param("id")
	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req EditUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	updatedUser, err := h.service.UpdateUser(uid, req.Name, req.Email, req.Role, req.IsActive)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        updatedUser.ID,
		"name":      updatedUser.Name,
		"email":     updatedUser.Email,
		"role":      updatedUser.Role,
		"is_active": updatedUser.IsActive,
	})
}

// ========================
// DELETE USER
// ========================
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.service.DeleteUser(uid); err != nil {
		utils.DeleteConflictResponse(c, err, "user")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
