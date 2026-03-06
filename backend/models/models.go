package models

import (
	"time"

	"github.com/google/uuid"
)

/* =========================
   ROLE
========================= */

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleSupport  Role = "support"
	RoleCustomer Role = "customer"
)

/* =========================
   USER
========================= */

type User struct {
	ID                uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name              string     `json:"name" gorm:"type:varchar(100)"`
	Email             string     `json:"email" gorm:"uniqueIndex;not null"`
	Password          string     `json:"-" gorm:"not null"` // 🔒 NEVER expose
	Role              Role       `json:"role" gorm:"type:varchar(20);not null"`
	IsActive          bool       `json:"is_active" gorm:"default:true"`
	MustResetPassword bool       `json:"must_reset_password" gorm:"default:false"`
	CreatedBy         *uuid.UUID `json:"created_by,omitempty" gorm:"type:uuid;index"`

	TwoFAEnabled bool       `json:"two_fa_enabled" gorm:"column:two_fa_enabled;default:false"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`


    LastOTPVerifiedAt    *time.Time `json:"last_otp_verified_at,omitempty"`
    LastPasswordResetAt *time.Time `json:"last_password_reset_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

/* =========================
   REFRESH TOKEN
========================= */

type RefreshToken struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;index;not null"`
	Token     string    `json:"-" gorm:"uniqueIndex;not null"` // 🔒 never expose
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	IsRevoked bool      `json:"is_revoked" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`

	User User `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

/* =========================
   PASSWORD RESET TOKEN
========================= */

type PasswordResetToken struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	Token     string    `json:"-" gorm:"uniqueIndex;not null"` // 🔒 never expose
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	Used      bool      `json:"used" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}

func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}

/* =========================
   2FA OTP
========================= */

type TwoFAOTP struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;index"`
	Code      string    `json:"-"` // 🔒 never expose
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

func (TwoFAOTP) TableName() string {
	return "two_fa_otps"
}
