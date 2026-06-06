// Command api is the HTTP entry point for the clinics Go backend. It wires
// configuration, the MySQL/GORM connection, and the Gin router together
// and serves the strict-server handlers generated from api/openapi.yaml.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/shahensargsyan/my-new-go-api/internal/api"
	"github.com/shahensargsyan/my-new-go-api/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := openDB(cfg.DB.DSN(), cfg.App.Debug)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	router := newRouter(cfg.App.Debug)
	jwtTTL := time.Duration(cfg.JWT.TTL) * time.Minute
	api.NewServer(db, []byte(cfg.JWT.Secret), jwtTTL).Register(router)

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("clinics-api listening on %s (env=%s, debug=%t)", addr, cfg.App.Env, cfg.App.Debug)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Print("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	log.Print("clinics-api stopped")
}

// openDB opens the GORM connection and tunes the underlying *sql.DB pool.
// Pool defaults are conservative — bump MaxOpenConns once we have a real
// load profile. Lifetime is bounded so connections don't outlive MySQL's
// wait_timeout (default 8h).
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
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}

// newRouter builds the bare Gin engine. The strict server's
// RegisterHandlers is responsible for the operation routes; everything
// else (health, docs, future static/asset serving) is wired here.
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
	// stock Swagger UI loaded from a CDN and pointed at /openapi.json —
	// no extra Go deps required. Both are public (no auth) so the docs
	// are reachable for anyone exploring the API.
	r.GET("/openapi.json", func(c *gin.Context) {
		spec, err := api.GetSwagger()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		// Rewrite servers to the host actually serving this docs page so
		// Swagger UI's "Try it out" hits us instead of whatever the spec
		// was authored with. Survives port changes, host header rewrites,
		// and future reverse-proxy fronting.
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
