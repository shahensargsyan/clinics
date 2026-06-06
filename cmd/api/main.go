// Command api is the local / container HTTP entry point for the clinics Go
// backend. It builds the application stack via internal/app and runs it as
// a long-lived server with graceful shutdown. The Vercel serverless
// deployment uses the same internal/app stack via api/index.go instead.
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

	"github.com/shahensargsyan/my-new-go-api/internal/app"
)

func main() {
	built, err := app.Build()
	if err != nil {
		log.Fatalf("%v", err)
	}
	cfg := built.Config

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           built.Engine,
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
	if sqlDB, err := built.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
	log.Print("clinics-api stopped")
}
