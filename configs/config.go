package configs

import (
	"fmt"
	"os"
)

// Config holds all configuration for the application
type Config struct {
	// Server
	ServerPort      string
	FrontendBaseURL string

	// Database
	DatabaseDSN string

	// Redis
	RedisURL string

	// RustFS/S3
	RustFSEndpoint           string
	RustFSAccessKey          string
	RustFSSecretKey          string
	RustFSRegion             string
	RustFSVideoBucket        string
	RustFSFilesBucket        string
	RustFSCertificatesBucket string
	RustFSBooksBucket        string

	// JWT
	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
	JWTIssuer         string

	// SMTP
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	// Admin development bypass
	AdminDevBypass bool
	AdminDevOTP    string

	// OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string

	MicrosoftClientID       string
	MicrosoftClientSecret   string
	MicrosoftRedirectURL    string
	OAuthTokenEncryptionKey string

	// Typesense
	TypesenseHost   string
	TypesensePort   string
	TypesenseAPIKey string

	// bKash
	BkashBaseURL     string
	BkashAppKey      string
	BkashAppSecret   string
	BkashUsername    string
	BkashPassword    string
	BkashCallbackURL string
}

// Load loads configuration from environment variables
// Fails fast on missing required variables
func Load() (*Config, error) {
	cfg := &Config{
		ServerPort:      getEnvOrDefault("SERVER_PORT", "8080"),
		FrontendBaseURL: getEnvOrDefault("FRONTEND_BASE_URL", "http://localhost:5173"),
		DatabaseDSN:     mustGetEnv("DATABASE_DSN"),
		RedisURL:        mustGetEnv("REDIS_URL"),

		RustFSEndpoint:           mustGetEnv("RUSTFS_ENDPOINT"),
		RustFSAccessKey:          mustGetEnv("RUSTFS_ACCESS_KEY"),
		RustFSSecretKey:          mustGetEnv("RUSTFS_SECRET_KEY"),
		RustFSRegion:             getEnvOrDefault("RUSTFS_REGION", "us-east-1"),
		RustFSVideoBucket:        getEnvOrDefault("RUSTFS_VIDEO_BUCKET", "lms-videos"),
		RustFSFilesBucket:        getEnvOrDefault("RUSTFS_FILES_BUCKET", "lms-files"),
		RustFSCertificatesBucket: getEnvOrDefault("RUSTFS_CERTIFICATES_BUCKET", "lms-certificates"),
		RustFSBooksBucket:        getEnvOrDefault("RUSTFS_BOOKS_BUCKET", "lms-books"),

		JWTPrivateKeyPath: mustGetEnv("JWT_PRIVATE_KEY_PATH"),
		JWTPublicKeyPath:  mustGetEnv("JWT_PUBLIC_KEY_PATH"),
		JWTIssuer:         getEnvOrDefault("JWT_ISSUER", "lms-backend"),

		SMTPHost:       os.Getenv("SMTP_HOST"),
		SMTPPort:       getEnvOrDefault("SMTP_PORT", "587"),
		SMTPUsername:   os.Getenv("SMTP_USERNAME"),
		SMTPPassword:   os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:       getEnvOrDefault("SMTP_FROM", "no-reply@example.com"),
		AdminDevBypass: getEnvOrDefault("ADMIN_DEV_BYPASS", "false") == "true",
		AdminDevOTP:    getEnvOrDefault("ADMIN_DEV_OTP", "000000"),

		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),

		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubRedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),

		MicrosoftClientID:       os.Getenv("MICROSOFT_CLIENT_ID"),
		MicrosoftClientSecret:   os.Getenv("MICROSOFT_CLIENT_SECRET"),
		MicrosoftRedirectURL:    os.Getenv("MICROSOFT_REDIRECT_URL"),
		OAuthTokenEncryptionKey: getEnvOrDefault("OAUTH_TOKEN_ENCRYPTION_KEY", getEnvOrDefault("JWT_ISSUER", "lms-backend")),

		TypesenseHost:   mustGetEnv("TYPESENSE_HOST"),
		TypesensePort:   getEnvOrDefault("TYPESENSE_PORT", "8108"),
		TypesenseAPIKey: mustGetEnv("TYPESENSE_API_KEY"),

		BkashBaseURL:     os.Getenv("BKASH_BASE_URL"),
		BkashAppKey:      os.Getenv("BKASH_APP_KEY"),
		BkashAppSecret:   os.Getenv("BKASH_APP_SECRET"),
		BkashUsername:    os.Getenv("BKASH_USERNAME"),
		BkashPassword:    os.Getenv("BKASH_PASSWORD"),
		BkashCallbackURL: os.Getenv("BKASH_CALLBACK_URL"),
	}

	return cfg, nil
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
