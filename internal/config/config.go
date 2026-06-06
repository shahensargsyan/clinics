// Package config loads runtime configuration from the shared Laravel `.env`
// file so the Go API can run side-by-side with the existing Laravel/Backpack
// project without divergent credentials.
package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"

	"github.com/joho/godotenv"
)

// Config is the top-level runtime configuration.
type Config struct {
	App   AppConfig
	DB    DBConfig
	JWT   JWTConfig
	Redis RedisConfig
}

type AppConfig struct {
	Name  string
	Env   string
	Key   string
	Debug bool
	URL   string
	Port  int // HTTP listener port; Go-service-only, not in the Laravel .env.
}

type DBConfig struct {
	Connection string
	Host       string
	Port       int
	Database   string
	Username   string
	Password   string
}

type JWTConfig struct {
	Secret string
	TTL    int // minutes; mirrors Laravel JWT_TTL
}

type RedisConfig struct {
	Host        string
	Port        int
	ForwardPort int
}

// DSN returns the GORM/MySQL DSN string in the format expected by
// gorm.io/driver/mysql.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Username, c.Password, c.Host, c.Port, c.Database,
	)
}

// Load reads the .env file and returns a populated Config. The env file
// defaults to `../laravel-doc/.env` (the sibling Laravel project) so this
// service shares one source of truth with the existing backend. Override
// with the CLINICS_ENV_FILE environment variable when running in
// production / CI where the env vars are injected directly.
func Load() (*Config, error) {
	envPath := os.Getenv("CLINICS_ENV_FILE")
	if envPath == "" {
		envPath = ".env"
	}

	switch _, err := os.Stat(envPath); {
	case err == nil:
		if err := godotenv.Load(envPath); err != nil {
			return nil, fmt.Errorf("config: load %s: %w", envPath, err)
		}
	case !os.IsNotExist(err):
		return nil, fmt.Errorf("config: stat %s: %w", envPath, err)
	}

	cfg := &Config{
		App: AppConfig{
			Name:  getEnv("APP_NAME", "clinics"),
			Env:   getEnv("APP_ENV", "development"),
			Key:   os.Getenv("APP_KEY"),
			Debug: getEnvBool("APP_DEBUG", false),
			URL:   getEnv("APP_URL", "http://localhost"),
			Port:  getEnvInt("HTTP_PORT", 8080),
		},
		DB: DBConfig{
			Connection: getEnv("DB_CONNECTION", "mysql"),
			Host:       getEnv("DB_HOST", "127.0.0.1"),
			Port:       getEnvInt("DB_PORT", 3306),
			Database:   os.Getenv("DB_DATABASE"),
			Username:   os.Getenv("DB_USERNAME"),
			Password:   os.Getenv("DB_PASSWORD"),
		},
		JWT: JWTConfig{
			Secret: os.Getenv("JWT_SECRET"),
			TTL:    getEnvInt("JWT_TTL", 60),
		},
		Redis: RedisConfig{
			Host:        getEnv("REDIS_HOST", "127.0.0.1"),
			Port:        getEnvInt("REDIS_PORT", 6379),
			ForwardPort: getEnvInt("FORWARD_REDIS_PORT", 0),
		},
	}

	if cfg.DB.Database == "" {
		return nil, fmt.Errorf("config: DB_DATABASE is required")
	}
	if cfg.DB.Username == "" {
		return nil, fmt.Errorf("config: DB_USERNAME is required")
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("config: JWT_SECRET is required")
	}

	cfg.DB.Host = resolveDBHost(cfg.DB.Host)
	return cfg, nil
}

// resolveDBHost handles the common dev-time case where DB_HOST in the shared
// .env points at a Docker Compose service name (e.g. "mysql-doc"). When this
// Go binary runs natively on macOS, that hostname will not resolve, so we
// transparently substitute 127.0.0.1 — the address the host uses when MySQL
// is published on its default port by docker-compose. On Linux (CI, prod,
// inside-container) the original hostname is preserved.
func resolveDBHost(host string) string {
	if host == "" || host == "localhost" || host == "127.0.0.1" {
		return host
	}
	if _, err := net.LookupHost(host); err == nil {
		return host
	}
	if runtime.GOOS == "darwin" {
		log.Printf("config: DB_HOST %q does not resolve on darwin; falling back to 127.0.0.1", host)
		return "127.0.0.1"
	}
	return host
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
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("config: %s=%q is not an int; using %d", key, v, fallback)
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Printf("config: %s=%q is not a bool; using %v", key, v, fallback)
		return fallback
	}
	return b
}
