package api

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Server is the strict-server implementation. Per-resource handler files
// (patients.go, ...) attach methods to *Server. The Server owns the GORM
// handle and is concurrency-safe to share across goroutines (GORM's *DB
// itself is safe for concurrent use).
type Server struct {
	DB *gorm.DB
}

// Compile-time assertion that *Server satisfies the generated interface.
var _ StrictServerInterface = (*Server)(nil)

func NewServer(db *gorm.DB) *Server { return &Server{DB: db} }

// Register installs the strict handler onto a Gin router. We wire up the
// HandlerErrorFunc so GORM's record-not-found becomes a 404 globally —
// this is the "global error handler middleware" referenced in the plan.
// Going through the strict-server's error funnel (instead of a separate
// Gin middleware) is necessary because the typed handler-returned errors
// never reach c.Errors / the gin chain.
func (s *Server) Register(r gin.IRouter) {
	si := NewStrictHandlerWithOptions(s, nil, StrictGinServerOptions{
		HandlerErrorFunc: handlerErrorFunc,
	})
	RegisterHandlers(r, si)
}

func handlerErrorFunc(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, Error{Message: "Resource not found"})
	default:
		log.Printf("api: handler error: %v", err)
		c.JSON(http.StatusInternalServerError, Error{Message: "Internal server error"})
	}
}
