package config

import (
	"fmt"
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
	RunInProcessCrons bool   // When false, SLA/contract crons run in worker containers
	CookieSameSite    string // none | lax | strict — use none for cross-origin SPA (Vercel ↔ API)
}

type DatabaseConfig struct {
	URL          string
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLife  time.Duration
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
	rememberSecret := os.Getenv("REMEMBER_DEVICE_SECRET")
	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:5173")
	storageType := getEnv("STORAGE_TYPE", "local")

	if env == "production" {
		mustRejectPlaceholder("JWT_ACCESS_SECRET", accessSecret, "access-secret", "CHANGE_ME")
		mustRejectPlaceholder("JWT_REFRESH_SECRET", refreshSecret, "refresh-secret", "CHANGE_ME")
		if accessSecret == "" || refreshSecret == "" {
			log.Fatal("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET are required when APP_ENV=production")
		}
		if dbURL == "" {
			log.Fatal("DATABASE_URL is required when APP_ENV=production")
		}
		if rememberSecret == "" || strings.Contains(rememberSecret, "CHANGE_ME") {
			log.Fatal("REMEMBER_DEVICE_SECRET is required when APP_ENV=production")
		}
		if frontendURL == "" || frontendURL == "http://localhost:5173" || strings.Contains(frontendURL, "your-app.vercel.app") || strings.Contains(frontendURL, "yourdomain.com") {
			log.Fatal("FRONTEND_URL must be set to your real Vercel/CRM origin when APP_ENV=production")
		}
		if storageType == "s3" {
			if getEnv("AWS_ACCESS_KEY_ID", "") == "" || getEnv("AWS_SECRET_ACCESS_KEY", "") == "" || getEnv("AWS_S3_BUCKET", "") == "" {
				log.Fatal("AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, and AWS_S3_BUCKET are required when STORAGE_TYPE=s3 in production")
			}
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

	// Cross-origin SPA (e.g. Vercel frontend + Lightsail API) needs SameSite=None + Secure
	// so the browser sends the refresh cookie on credentialed XHR after a page reload.
	cookieSameSite := strings.ToLower(strings.TrimSpace(getEnv("COOKIE_SAMESITE", "")))
	if cookieSameSite == "" {
		if env == "production" {
			cookieSameSite = "none"
		} else {
			cookieSameSite = "lax"
		}
	}
	switch cookieSameSite {
	case "none", "lax", "strict":
	default:
		log.Fatalf("COOKIE_SAMESITE must be none, lax, or strict (got %q)", cookieSameSite)
	}

	return &Config{
		Server: ServerConfig{
			Port:              getEnv("SERVER_PORT", "8080"),
			Env:               env,
			RateLimitMax:      getEnvAsInt("RATE_LIMIT_MAX", 60),
			TrustedProxies:    getEnvCSV("TRUSTED_PROXIES", []string{"nginx", "172.16.0.0/12", "10.0.0.0/8"}),
			RunInProcessCrons: getEnvAsBool("RUN_INPROCESS_CRONS", true),
			CookieSameSite:    cookieSameSite,
		},
		Database: DatabaseConfig{
			URL:          dbURL,
			MaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 3),
			ConnMaxLife:  time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute,
		},
		JWT: JWTConfig{
			AccessSecret:  accessSecret,
			RefreshSecret: refreshSecret,
			AccessExpiry:  time.Duration(getEnvAsInt("JWT_ACCESS_EXPIRY_MINUTES", 15)) * time.Minute,
			RefreshExpiry: time.Duration(getEnvAsInt("JWT_REFRESH_EXPIRY_DAYS", 7)) * 24 * time.Hour,
		},
		FrontendURL: frontendURL,

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
			Type:     storageType,
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

func mustRejectPlaceholder(name, value string, forbidden ...string) {
	for _, f := range forbidden {
		if value == f || strings.Contains(value, f) {
			log.Fatal(fmt.Sprintf("refusing to start with placeholder %s in production", name))
		}
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
