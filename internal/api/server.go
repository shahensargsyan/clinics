package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Server is the strict-server implementation. Per-resource handler files
// (patients.go, auth.go, ...) attach methods to *Server. The Server owns
// the GORM handle plus the JWT signing material; it is concurrency-safe
// to share across goroutines.
type Server struct {
	DB        *gorm.DB
	jwtSecret []byte
	jwtTTL    time.Duration
}

var _ StrictServerInterface = (*Server)(nil)

func NewServer(db *gorm.DB, jwtSecret []byte, jwtTTL time.Duration) *Server {
	return &Server{DB: db, jwtSecret: jwtSecret, jwtTTL: jwtTTL}
}

// Register installs the strict handler onto a Gin router. The strict
// middleware chain handles bearer-token enforcement (authMiddleware) and
// the typed-error funnel (handlerErrorFunc) maps GORM/auth sentinels to
// HTTP status codes.
func (s *Server) Register(r gin.IRouter) {
	si := NewStrictHandlerWithOptions(s,
		[]StrictMiddlewareFunc{s.authMiddleware},
		StrictGinServerOptions{HandlerErrorFunc: handlerErrorFunc},
	)
	RegisterHandlers(r, si)
}

func handlerErrorFunc(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errUnauthorized):
		c.JSON(http.StatusUnauthorized, Error{Message: "Unauthorized"})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, Error{Message: "Resource not found"})
	default:
		log.Printf("api: handler error: %v", err)
		c.JSON(http.StatusInternalServerError, Error{Message: "Internal server error"})
	}
}
