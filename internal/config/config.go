// Package config loads runtime configuration from the shared Laravel `.env`
// file so the Go API can run side-by-side with the existing Laravel/Backpack
// project without divergent credentials.
package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	gomysql "github.com/go-sql-driver/mysql"
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

	// SSLMode controls how MySQL TLS is wired:
	//   ""            no TLS (default).
	//   "true"        TLS using the OS trust store; hostname IS checked.
	//                 No DB_SSLROOTCERT needed. Good fit for managed DBs
	//                 with publicly-signed certs (RDS, PlanetScale, etc).
	//   "verify-ca"   chain verified against a custom CA from
	//                 DB_SSLROOTCERT; hostname NOT checked. Matches
	//                 libpq's vocabulary.
	//   "verify-full" custom CA + hostname check.
	SSLMode string
	// SSLRootCert is either a filesystem path to a PEM file OR the raw
	// PEM-encoded CA bundle itself (anything starting with `-----BEGIN`).
	// Inline form is intended for serverless platforms that can't ship a
	// sidecar file but can inject multi-line env vars. Required when
	// SSLMode is "verify-ca" or "verify-full"; ignored otherwise.
	SSLRootCert string
}

// tlsConfigName is the key we register the loaded *tls.Config under with
// go-sql-driver/mysql. The DSN references it as `tls=<this-name>`.
const tlsConfigName = "clinics-tls"

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
// gorm.io/driver/mysql. When SSLMode is set the DSN points at a TLS
// config that must already be registered — callers must invoke
// RegisterTLS() before gorm.Open(). DSN itself stays a pure formatter
// (no IO, no side effects) so it remains safe to call repeatedly.
func (c DBConfig) DSN() string {
	// Three separate timeouts — all are required to guarantee a
	// fast-failing connection on serverless:
	//   timeout      → TCP dial only. Does NOT cover the TLS or auth
	//                  handshake. A reachable-but-misbehaving host (e.g.
	//                  a TLS handshake stall against Aiven's strict SSL)
	//                  passes the dial and then hangs here forever
	//                  without the next two.
	//   readTimeout  → bounds every read, INCLUDING the server's TLS +
	//                  auth handshake packets. This is what converts a
	//                  30s Vercel function-timeout into a clean, fast Go
	//                  error.
	//   writeTimeout → bounds every write, including the client handshake.
	// 10s is well under typical serverless function ceilings yet generous
	// enough not to kill healthy queries in this admin workload.
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local"+
			"&timeout=5s&readTimeout=10s&writeTimeout=10s",
		c.Username, c.Password, c.Host, c.Port, c.Database,
	)
	switch c.SSLMode {
	case "":
		// plain TCP
	case "true":
		// Driver's built-in TLS: system trust store + hostname check.
		dsn += "&tls=true"
	default:
		// "verify-ca" / "verify-full" — points at the *tls.Config that
		// RegisterTLS() must have installed for this name.
		dsn += "&tls=" + tlsConfigName
	}
	return dsn
}

// RegisterTLS reads the CA root certificate referenced by SSLRootCert
// and registers a *tls.Config with go-sql-driver/mysql under the name
// embedded in DSN(). Must be called before gorm.Open() when SSLMode is
// set. No-op when SSLMode is empty.
func (c DBConfig) RegisterTLS() error {
	// "" → no TLS at all. "true" → driver handles TLS using OS roots
	// and needs no app-side registration. Only verify-ca / verify-full
	// build a custom config from DB_SSLROOTCERT.
	if c.SSLMode == "" || c.SSLMode == "true" {
		return nil
	}
	caPEM, source, err := loadCABundle(c.SSLRootCert)
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return fmt.Errorf("config: CA bundle from %s contained no PEM certificates", source)
	}
	tlsCfg := &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	switch c.SSLMode {
	case "verify-full":
		tlsCfg.ServerName = c.Host
	case "verify-ca":
		// libpq verify-ca: chain must validate, hostname is NOT checked.
		// The Go stdlib bundles those two checks together, so we disable
		// its automatic verification and run chain-only verification by
		// hand via VerifyPeerCertificate.
		tlsCfg.InsecureSkipVerify = true
		tlsCfg.VerifyPeerCertificate = verifyChainOnly(pool)
	}
	return gomysql.RegisterTLSConfig(tlsConfigName, tlsCfg)
}

// loadCABundle returns the PEM bytes for the CA bundle along with a
// human-readable source label for error messages. It auto-detects
// whether the input is inline PEM (starts with `-----BEGIN`) or a path
// to a file containing PEM. Inline is the serverless-friendly form.
func loadCABundle(s string) (pem []byte, source string, err error) {
	if strings.HasPrefix(strings.TrimSpace(s), "-----BEGIN") {
		return []byte(s), "inline DB_SSLROOTCERT value", nil
	}
	b, err := os.ReadFile(s)
	if err != nil {
		return nil, "", fmt.Errorf("config: read CA cert %q: %w", s, err)
	}
	return b, fmt.Sprintf("file %q", s), nil
}

// verifyChainOnly returns a tls.Config.VerifyPeerCertificate that
// validates the presented chain against `pool` but skips the hostname
// match, matching libpq's `sslmode=verify-ca` semantics.
func verifyChainOnly(pool *x509.CertPool) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return fmt.Errorf("tls: server presented no certificate")
		}
		certs := make([]*x509.Certificate, 0, len(rawCerts))
		for _, raw := range rawCerts {
			cert, err := x509.ParseCertificate(raw)
			if err != nil {
				return err
			}
			certs = append(certs, cert)
		}
		opts := x509.VerifyOptions{Roots: pool}
		if len(certs) > 1 {
			opts.Intermediates = x509.NewCertPool()
			for _, intermediate := range certs[1:] {
				opts.Intermediates.AddCert(intermediate)
			}
		}
		_, err := certs[0].Verify(opts)
		return err
	}
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
			Connection:  getEnv("DB_CONNECTION", "mysql"),
			Host:        getEnv("DB_HOST", "127.0.0.1"),
			Port:        getEnvInt("DB_PORT", 3306),
			Database:    os.Getenv("DB_DATABASE"),
			Username:    os.Getenv("DB_USERNAME"),
			Password:    os.Getenv("DB_PASSWORD"),
			SSLMode:     getEnv("DB_SSLMODE", ""),
			SSLRootCert: getEnv("DB_SSLROOTCERT", ""),
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
	switch cfg.DB.SSLMode {
	case "", "true", "verify-ca", "verify-full":
		// OK.
	default:
		return nil, fmt.Errorf("config: DB_SSLMODE must be one of '', 'true', 'verify-ca', 'verify-full' (got %q)", cfg.DB.SSLMode)
	}
	// DB_SSLROOTCERT is only meaningful for the custom-CA modes; "true"
	// uses the OS trust store so the cert env var is irrelevant.
	if (cfg.DB.SSLMode == "verify-ca" || cfg.DB.SSLMode == "verify-full") && cfg.DB.SSLRootCert == "" {
		return nil, fmt.Errorf("config: DB_SSLROOTCERT is required when DB_SSLMODE=%s", cfg.DB.SSLMode)
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
