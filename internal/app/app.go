// Package app wires configuration, the MySQL/GORM connection, and the Gin
// router into a ready-to-serve http.Handler. It is the single source of
// truth for application bootstrap so both entrypoints — the local
// long-running server (cmd/api) and the Vercel serverless function
// (api/index.go) — build the exact same stack.
package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/shahensargsyan/my-new-go-api/internal/api"
	"github.com/shahensargsyan/my-new-go-api/internal/config"
)

// Built is everything Build() produces. cmd/api needs all three (port,
// graceful-shutdown close); the serverless handler only needs Engine.
type Built struct {
	Engine *gin.Engine
	Config *config.Config
	DB     *gorm.DB
}

// Build loads config, registers TLS, opens the pooled MySQL connection,
// and returns the fully-wired Gin engine. It performs a Ping, so a return
// without error means the database is reachable. Safe to call once per
// process (the serverless handler caches the result across warm
// invocations).
func Build() (*Built, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := cfg.DB.RegisterTLS(); err != nil {
		return nil, fmt.Errorf("db tls: %w", err)
	}
	db, err := openDB(cfg.DB.DSN(), cfg.App.Debug)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	engine := newRouter(cfg.App.Debug)
	jwtTTL := time.Duration(cfg.JWT.TTL) * time.Minute
	api.NewServer(db, []byte(cfg.JWT.Secret), jwtTTL).Register(engine)
	return &Built{Engine: engine, Config: cfg, DB: db}, nil
}

// openDB opens the GORM connection and tunes the underlying *sql.DB pool
// with serverless-safe bounds (see comments inline).
func openDB(dsn string, debug bool) (*gorm.DB, error) {
	logMode := gormlogger.Warn
	if debug {
		logMode = gormlogger.Info
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logMode),
	})
	if err != nil {
		return nil, fmt.Errorf("gorm.Open: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("db.DB: %w", err)
	}
	// Serverless-safe pool bounds. On Vercel each concurrent invocation
	// is its own process; a high per-instance cap multiplied across
	// horizontal scale-out would blow past Aiven's connection limit.
	// Keep each instance frugal: at most 1 idle + 5 open.
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(5)
	// Cap idle lifetime tight so a frozen/thawed serverless instance
	// doesn't reuse a connection Aiven already reaped server-side.
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Minute)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}

// newRouter builds the Gin engine: health, docs, and (via RegisterHandlers
// inside Build) the generated API routes.
func newRouter(debug bool) *gin.Engine {
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// /openapi.json serves the spec embedded in openapi.gen.go. /docs is a
	// stock Swagger UI loaded from a CDN and pointed at /openapi.json.
	r.GET("/openapi.json", func(c *gin.Context) {
		spec, err := api.GetSwagger()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		// Rewrite servers to the host actually serving this docs page so
		// Swagger UI's "Try it out" hits us, not the authored host.
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		spec.Servers = openapi3.Servers{{
			URL:         fmt.Sprintf("%s://%s", scheme, c.Request.Host),
			Description: "Current host",
		}}
		c.JSON(http.StatusOK, spec)
	})
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML))
	})

	return r
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Clinics API Docs</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
  <style>body{margin:0}</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: '/openapi.json',
        dom_id: '#swagger-ui',
        deepLinking: true,
        persistAuthorization: true
      });
    };
  </script>
</body>
</html>`
