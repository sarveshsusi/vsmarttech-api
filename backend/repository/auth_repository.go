package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/models"
)

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

/* =====================
   User Queries
===================== */

func (r *AuthRepository) FindUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.
		Where("email = ? AND is_active = true", email).
		First(&user).
		Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *AuthRepository) FindUserByID(id uuid.UUID) (*models.User, error) {
	var user models.User

	err := r.db.
		Where("id = ? AND is_active = true", id).
		First(&user).
		Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByID fetches a user by ID regardless of active status
// Used for admin operations like editing user details and status
func (r *AuthRepository) GetUserByID(id uuid.UUID) (*models.User, error) {
	var user models.User

	err := r.db.
		Where("id = ?", id).
		First(&user).
		Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *AuthRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}
func (r *AuthRepository) CreateUserTx(
	tx *gorm.DB,
	user *models.User,
) error {
	return tx.Create(user).Error
}

func (r *AuthRepository) UpdateUserPassword(
	userID uuid.UUID,
	hashedPassword string,
	mustReset bool,
) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password":            hashedPassword,
			"must_reset_password": mustReset,
		}).Error
}

/* =====================
   Refresh Tokens
===================== */

func (r *AuthRepository) CreateRefreshToken(rt *models.RefreshToken) error {
	return r.db.Create(rt).Error
}

func (r *AuthRepository) RevokeAllUserTokens(userID uuid.UUID) error {
	return r.db.Model(&models.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true).Error
}

func (r *AuthRepository) FindRefreshToken(tokenHash string) (*models.RefreshToken, error) {
	var rt models.RefreshToken

	err := r.db.
		Preload("User").
		Where(
			"token = ? AND is_revoked = false AND expires_at > ?",
			tokenHash,
			time.Now(),
		).
		First(&rt).
		Error

	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}

	return &rt, nil
}

func (r *AuthRepository) RevokeRefreshToken(tokenHash string) error {
	result := r.db.Model(&models.RefreshToken{}).
		Where("token = ? AND is_revoked = false", tokenHash).
		Update("is_revoked", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("refresh token already revoked or not found")
	}

	return nil
}

/* =====================
   Password Reset
===================== */

func (r *AuthRepository) CreatePasswordReset(
	reset *models.PasswordResetToken,
) error {
	return r.db.Create(reset).Error
}

func (r *AuthRepository) FindValidPasswordReset(
	hashedToken string,
) (*models.PasswordResetToken, error) {
	var reset models.PasswordResetToken

	err := r.db.
		Where("token = ? AND used = false AND expires_at > NOW()", hashedToken).
		First(&reset).Error

	if err != nil {
		return nil, err
	}

	return &reset, nil
}

func (r *AuthRepository) MarkPasswordResetUsed(id uuid.UUID) error {
	return r.db.
		Model(&models.PasswordResetToken{}).
		Where("id = ?", id).
		Update("used", true).
		Error
}

/* =====================
   2FA OTP
===================== */

func (r *AuthRepository) Create2FAOTP(otp *models.TwoFAOTP) error {
	return r.db.Create(otp).Error
}

func (r *AuthRepository) FindValid2FAOTP(
	userID uuid.UUID,
	code string,
) (*models.TwoFAOTP, error) {
	var otp models.TwoFAOTP

	err := r.db.Where(
		"user_id = ? AND code = ? AND used = false AND expires_at > NOW()",
		userID,
		code,
	).First(&otp).Error

	if err != nil {
		return nil, err
	}

	return &otp, nil
}

func (r *AuthRepository) MarkOTPUsed(id uuid.UUID) error {
	return r.db.Model(&models.TwoFAOTP{}).
		Where("id = ?", id).
		Update("used", true).
		Error
}

func (r *AuthRepository) MarkAllOTPUsed(userID uuid.UUID) error {
	return r.db.Model(&models.TwoFAOTP{}).
		Where("user_id = ? AND used = false", userID).
		Update("used", true).
		Error
}

/* =====================
   2FA Settings
===================== */

func (r *AuthRepository) Enable2FA(userID uuid.UUID) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("two_fa_enabled", true).Error
}

func (r *AuthRepository) Disable2FA(userID uuid.UUID) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("two_fa_enabled", false).Error
}

/* =====================
   Audit
===================== */

func (r *AuthRepository) UpdateLastLogin(
	userID uuid.UUID,
	t *time.Time,
) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("last_login_at", t).Error
}

/* =====================
   Admin: Get Users (Paginated)
===================== */

func (r *AuthRepository) GetUsersPaginated(
	limit int,
	offset int,
) ([]models.User, int64, error) {

	var users []models.User
	var total int64

	// Count total users (including inactive)
	if err := r.db.
		Model(&models.User{}).
		Count(&total).
		Error; err != nil {
		return nil, 0, err
	}

	// Fetch paginated users (all users, regardless of status)
	query := r.db

	if err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).
		Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *AuthRepository) GetUsersByRole(role models.Role) ([]models.User, error) {
	var users []models.User
	err := r.db.Where("role = ?", role).
		Order("name ASC").
		Find(&users).Error
	return users, err
}

func (r *AuthRepository) CreateRememberedDevice(rd *models.RememberedDevice) error {
	return r.db.Create(rd).Error
}

func (r *AuthRepository) FindValidRememberedDevice(
	userID uuid.UUID,
	hashedToken string,
) (*models.RememberedDevice, error) {

	var rd models.RememberedDevice

	err := r.db.Where(
		"user_id = ? AND token = ? AND expires_at > NOW()",
		userID,
		hashedToken,
	).First(&rd).Error

	if err != nil {
		return nil, err
	}

	return &rd, nil
}

func (r *AuthRepository) DeleteUserDevices(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).
		Delete(&models.RememberedDevice{}).Error
}

func (r *AuthRepository) ErrNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
