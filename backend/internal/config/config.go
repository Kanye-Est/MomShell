package config

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DatabaseURL string

	// JWT
	JWTSecretKey string
	JWTAlgorithm string
	ImageModel   string

	mu      sync.RWMutex
	runtime RuntimeSettings

	// Firecrawl (web search)
	FirecrawlAPIKey string

	// Server
	Port           string
	TrustedProxies []string

	// CORS
	CORSOrigins string

	// Logging
	DBLogLevel string

	// Admin
	AdminUsername string
	AdminEmail    string
	AdminPassword string
}

type RuntimeSettings struct {
	JWTAccessTokenExpireMin   int
	JWTRefreshTokenExpireDays int
	OpenAIAPIKey              string
	OpenAIBaseURL             string
	OpenAIModel               string
}

func Load() *Config {
	// The process environment takes precedence over the project and backend files.
	if _, err := os.Stat("go.mod"); err == nil {
		_ = godotenv.Load("../.env")
	}
	_ = godotenv.Load()

	secret := getEnv("JWT_SECRET_KEY", "")
	if secret == "" || secret == "change-me-in-production" {
		secret = rand.Text()
		log.Println("[WARN] Generated a temporary JWT secret. Set JWT_SECRET_KEY to preserve login sessions across restarts.")
	}

	proxies := strings.FieldsFunc(os.Getenv("TRUSTED_PROXIES"), func(r rune) bool { return r == ',' })
	for i, proxy := range proxies {
		proxies[i] = strings.TrimSpace(proxy)
	}
	imageModel, configured := os.LookupEnv("IMAGE_MODEL")
	if !configured {
		imageModel = "Tongyi-MAI/Z-Image-Turbo"
	}

	return &Config{
		DatabaseURL:  getEnv("DATABASE_URL", ""),
		JWTSecretKey: secret,
		JWTAlgorithm: "HS256",
		ImageModel:   strings.TrimSpace(imageModel),
		runtime: RuntimeSettings{
			JWTAccessTokenExpireMin:   getEnvInt("JWT_ACCESS_TOKEN_EXPIRE_MINUTES", 30),
			JWTRefreshTokenExpireDays: getEnvInt("JWT_REFRESH_TOKEN_EXPIRE_DAYS", 7),
			OpenAIAPIKey:              getEnv("OPENAI_API_KEY", ""),
			OpenAIBaseURL:             getEnv("OPENAI_BASE_URL", "https://api-inference.modelscope.cn/v1"),
			OpenAIModel:               getEnv("OPENAI_MODEL", "Qwen/Qwen3-235B-A22B"),
		},
		FirecrawlAPIKey: getEnv("FIRECRAWL_API_KEY", ""),
		Port:            getEnv("PORT", "8000"),
		TrustedProxies:  proxies,
		CORSOrigins:     getEnv("CORS_ORIGINS", "*"),
		DBLogLevel:      getEnv("DB_LOG_LEVEL", "warn"),
		AdminUsername:   getEnv("ADMIN_USERNAME", ""),
		AdminEmail:      getEnv("ADMIN_EMAIL", ""),
		AdminPassword:   getEnv("ADMIN_PASSWORD", ""),
	}
}

func (c *Config) Runtime() RuntimeSettings {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.runtime
}

// UpdateRuntime applies a configuration change to the current server process.
func (c *Config) UpdateRuntime(items map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	next := c.runtime
	for key, raw := range items {
		value := strings.TrimSpace(raw)
		switch key {
		case "OPENAI_API_KEY":
			next.OpenAIAPIKey = value
		case "OPENAI_BASE_URL":
			endpoint, err := url.Parse(value)
			if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
				return fmt.Errorf("OPENAI_BASE_URL 必须是 HTTP 或 HTTPS 地址")
			}
			next.OpenAIBaseURL = value
		case "OPENAI_MODEL":
			next.OpenAIModel = value
		case "JWT_ACCESS_TOKEN_EXPIRE_MINUTES", "JWT_REFRESH_TOKEN_EXPIRE_DAYS":
			duration, err := strconv.Atoi(value)
			if err != nil || duration <= 0 {
				return fmt.Errorf("%s 必须是正整数", key)
			}
			if key == "JWT_ACCESS_TOKEN_EXPIRE_MINUTES" {
				next.JWTAccessTokenExpireMin = duration
			} else {
				next.JWTRefreshTokenExpireDays = duration
			}
		default:
			return fmt.Errorf("配置项 %s 不可编辑", key)
		}
	}
	c.runtime = next
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil || i <= 0 {
		return fallback
	}
	return i
}
