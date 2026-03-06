package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	JWT         JWTConfig
	FrontendURL string
	Mail        MailConfig
	ImageKit    ImageKitConfig // ✅ KEPT for backward compatibility
	Storage     StorageConfig
	AWS         AWSConfig
	Image       ImageConfig
}

type ServerConfig struct {
	Port         string
	Env          string
	RateLimitMax int // Max requests per minute
}

type DatabaseConfig struct {
	URL string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

/* =====================
   Mail (SMTP / Mailtrap)
===================== */

type MailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

/* =====================
   ImageKit (CDN Uploads)
===================== */

type ImageKitConfig struct {
	PublicKey  string
	PrivateKey string
	Endpoint   string
}

/* =====================
   Storage Configuration
===================== */

type StorageConfig struct {
	Type     string // "local" or "s3"
	LocalDir string // For local storage
	BaseURL  string // For local storage
}

/* =====================
   AWS S3 Configuration
===================== */

type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Folder          string
}

/* =====================
   Image Configuration
===================== */

type ImageConfig struct {
	MaxSizeBytes           int64
	CompressThresholdBytes int64
	Quality                int
}

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Env:          getEnv("APP_ENV", "development"),
			RateLimitMax: getEnvAsInt("RATE_LIMIT_MAX", 1000), // 1000 requests/min for dev, use env to override
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "access-secret"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "refresh-secret"),
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),

		Mail: MailConfig{
			Host:     getEnv("MAIL_HOST", ""),
			Port:     getEnvAsInt("MAIL_PORT", 587),
			Username: getEnv("MAIL_USERNAME", ""),
			Password: getEnv("MAIL_PASSWORD", ""),
			From:     getEnv("MAIL_FROM", "rbac@app.com"),
		},

		ImageKit: ImageKitConfig{
			PublicKey:  getEnv("IMAGEKIT_PUBLIC_KEY", ""),
			PrivateKey: getEnv("IMAGEKIT_PRIVATE_KEY", ""),
			Endpoint:   getEnv("IMAGEKIT_URL_ENDPOINT", ""),
		},

		Storage: StorageConfig{
			Type:     getEnv("STORAGE_TYPE", "local"), // "local" for development, "s3" for production
			LocalDir: getEnv("STORAGE_LOCAL_DIR", "./uploads"),
			BaseURL:  getEnv("STORAGE_BASE_URL", "http://localhost:8080/uploads"),
		},

		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", "ap-south-1"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			Bucket:          getEnv("AWS_S3_BUCKET", ""),
			Folder:          getEnv("AWS_S3_FOLDER", "uploads"),
		},

		Image: ImageConfig{
			MaxSizeBytes:           getEnvAsInt64("IMAGE_MAX_SIZE_BYTES", 1048576),         // 1 MB
			CompressThresholdBytes: getEnvAsInt64("IMAGE_COMPRESS_THRESHOLD_BYTES", 51200), // 50 KB
			Quality:                getEnvAsInt("IMAGE_QUALITY", 85),
		},
	}
}

/* =====================
   Helpers
===================== */

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvAsInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}
