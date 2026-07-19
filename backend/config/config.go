package config

import (
	"log"
	"os"
	"strconv"
	"strings"
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
	Port              string
	Env               string
	RateLimitMax      int // Max requests per minute
	TrustedProxies    []string
	RunInProcessCrons bool // When false, SLA/contract crons run in worker containers
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
	env := getEnv("APP_ENV", "development")
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")

	if env == "production" {
		if accessSecret == "" || refreshSecret == "" {
			log.Fatal("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET are required when APP_ENV=production")
		}
		if accessSecret == "access-secret" || refreshSecret == "refresh-secret" {
			log.Fatal("refusing to start with default JWT secrets in production")
		}
	} else {
		if accessSecret == "" {
			accessSecret = "access-secret"
			log.Println("warning: JWT_ACCESS_SECRET unset; using insecure development default")
		}
		if refreshSecret == "" {
			refreshSecret = "refresh-secret"
			log.Println("warning: JWT_REFRESH_SECRET unset; using insecure development default")
		}
	}

	return &Config{
		Server: ServerConfig{
			Port:              getEnv("SERVER_PORT", "8080"),
			Env:               env,
			RateLimitMax:      getEnvAsInt("RATE_LIMIT_MAX", 1000),
			TrustedProxies:    getEnvCSV("TRUSTED_PROXIES", []string{"nginx", "172.16.0.0/12", "10.0.0.0/8"}),
			RunInProcessCrons: getEnvAsBool("RUN_INPROCESS_CRONS", true),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		JWT: JWTConfig{
			AccessSecret:  accessSecret,
			RefreshSecret: refreshSecret,
			AccessExpiry:  time.Duration(getEnvAsInt("JWT_ACCESS_EXPIRY_MINUTES", 15)) * time.Minute,
			RefreshExpiry: time.Duration(getEnvAsInt("JWT_REFRESH_EXPIRY_DAYS", 7)) * 24 * time.Hour,
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

func getEnvAsBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvCSV(key string, fallback []string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
