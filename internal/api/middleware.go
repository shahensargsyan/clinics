package api

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

// publicOps lists strict-server operation IDs that bypass bearer auth.
// Operation IDs here use the PascalCase Go method names (what the
// generated strict-handler passes as `operationID`), not the camelCase
// IDs in the OpenAPI spec.
var publicOps = map[string]struct{}{
	"LoginUser": {},
}

// authMiddleware enforces the Authorization: Bearer <jwt> header on all
// operations not listed in publicOps. The decoded user id is stashed on
// both the gin.Context (via userIDContextKey) and the request context
// (via a custom key) so it's accessible from both interfaces.
//
// On failure the middleware returns errUnauthorized; handlerErrorFunc
// converts that to a 401 JSON response. Doing it via a typed error keeps
// the middleware decoupled from the dozens of per-operation
// *401JSONResponse types the generator emits.
func (s *Server) authMiddleware(next StrictHandlerFunc, operationID string) StrictHandlerFunc {
	return func(ctx *gin.Context, request any) (any, error) {
		if _, public := publicOps[operationID]; public {
			return next(ctx, request)
		}
		token, ok := bearerToken(ctx)
		if !ok {
			return nil, errUnauthorized
		}
		uid, err := s.validateToken(token)
		if err != nil {
			return nil, errUnauthorized
		}
		ctx.Set(userIDContextKey, uid)
		// Also store on the request context for handlers that receive context.Context
		ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), userIDContextKey, uid))
		return next(ctx, request)
	}
}

func bearerToken(ctx *gin.Context) (string, bool) {
	h := ctx.GetHeader("Authorization")
	if h == "" {
		return "", false
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}
