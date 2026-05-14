package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

// errUnauthorized is a sentinel returned by handlers and middleware when
// authentication fails. handlerErrorFunc translates it to a 401 JSON
// response so individual handlers don't have to know about typed
// per-operation response objects for the 401 case.
var errUnauthorized = errors.New("unauthorized")

// userIDContextKey is what the auth middleware sets on the gin.Context
// after a successful bearer-token validation. Handlers should fetch via
// UserIDFromContext rather than the raw key.
const userIDContextKey = "auth.userID"

// UserIDFromContext returns the authenticated user's id, or 0 + false if
// the request was unauthenticated (which should only happen for routes in
// publicOps).
func UserIDFromContext(ctx *gin.Context) (uint, bool) {
	v, ok := ctx.Get(userIDContextKey)
	if !ok {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok
}

// issueToken signs a short JWT for the given user. Claims kept minimal —
// "sub" (subject = user id) + standard iat/exp — so the token surface is
// small. Add custom claims when there's a concrete need.
func (s *Server) issueToken(userID uint) (token string, expiresAt time.Time, err error) {
	now := time.Now()
	expiresAt = now.Add(s.jwtTTL)
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": now.Unix(),
		"exp": expiresAt.Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = t.SignedString(s.jwtSecret)
	return
}

// validateToken parses and verifies a bearer token; returns the subject
// (user id) on success.
func (s *Server) validateToken(tokenStr string) (uint, error) {
	parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
		}
		return s.jwtSecret, nil
	})
	if err != nil || !parsed.Valid {
		return 0, errUnauthorized
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errUnauthorized
	}
	// JSON numbers unmarshal as float64 in MapClaims.
	sub, ok := claims["sub"].(float64)
	if !ok || sub <= 0 {
		return 0, errUnauthorized
	}
	return uint(sub), nil
}

// LoginUser implements the strict-server login handler. Behaviour mirrors
// the Laravel `Auth::guard('api')->attempt([username, password])` flow:
// look up the user by email (which the wire format calls `username` for
// Laravel compatibility), bcrypt-compare the password, and issue a JWT.
//
// Error responses intentionally do not distinguish "no such user" from
// "wrong password" — both surface as 401 to avoid user-enumeration leaks.
func (s *Server) LoginUser(ctx context.Context, req LoginUserRequestObject) (LoginUserResponseObject, error) {
	if req.Body == nil {
		return loginValidation422("Request body is required.", nil), nil
	}
	if errs := validateLoginRequest(*req.Body); len(errs) > 0 {
		return loginValidation422("The given data was invalid.", errs), nil
	}

	var u models.User
	err := s.DB.WithContext(ctx).Where("email = ?", req.Body.Username).First(&u).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return login401(), nil
	case err != nil:
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Body.Password)); err != nil {
		return login401(), nil
	}

	token, expiresAt, err := s.issueToken(u.ID)
	if err != nil {
		return nil, err
	}

	return LoginUser200JSONResponse{
		AccessToken: token,
		TokenType:   Bearer,
		ExpiresIn:   int(time.Until(expiresAt).Seconds()),
		User: AuthUser{
			Id:    int64(u.ID),
			Name:  u.Name,
			Email: openapi_types.Email(u.Email),
		},
	}, nil
}

func validateLoginRequest(b LoginRequest) map[string][]string {
	errs := map[string][]string{}
	if b.Username == "" {
		errs["username"] = []string{"The username field is required."}
	}
	if b.Password == "" {
		errs["password"] = []string{"The password field is required."}
	}
	return errs
}

func login401() LoginUser401JSONResponse {
	return LoginUser401JSONResponse{UnauthorizedJSONResponse{Message: "Invalid credentials"}}
}

func loginValidation422(message string, fieldErrs map[string][]string) LoginUser422JSONResponse {
	if fieldErrs == nil {
		fieldErrs = map[string][]string{}
	}
	return LoginUser422JSONResponse{ValidationErrorJSONResponse{
		Message: message,
		Errors:  fieldErrs,
	}}
}
