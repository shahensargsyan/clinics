// Package handler is the Vercel serverless entry point. Vercel's Go runtime
// invokes the exported Handler function for each request; it must not bind a
// port (that's what the local cmd/api server does instead).
//
// The Gin engine + pooled DB connection are built once and cached across
// warm invocations on the same instance — that's what keeps the
// per-instance pool (max 5) from being recreated on every request.
package handler

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/shahensargsyan/my-new-go-api/internal/app"
)

var (
	mu     sync.Mutex
	engine *gin.Engine
)

// Handler is the Vercel function entry point. Paths arrive unmodified
// (the vercel.json rewrite is transparent), so the Gin router matches
// /patients, /auth/login, etc. exactly as it does locally.
func Handler(w http.ResponseWriter, r *http.Request) {
	eng, err := getEngine()
	if err != nil {
		// Build failed (almost always: DB unreachable). Thanks to the
		// readTimeout/writeTimeout in the DSN this returns in ~10s with a
		// real message instead of hanging to the function's maxDuration.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "service initialization failed",
			"error":   err.Error(),
		})
		return
	}
	eng.ServeHTTP(w, r)
}

// getEngine lazily builds the app once and caches it only on success, so a
// transient DB outage during a cold start can be retried on the next
// invocation rather than poisoning the warm instance permanently.
func getEngine() (*gin.Engine, error) {
	mu.Lock()
	defer mu.Unlock()
	if engine != nil {
		return engine, nil
	}
	built, err := app.Build()
	if err != nil {
		return nil, err
	}
	engine = built.Engine
	return engine, nil
}
