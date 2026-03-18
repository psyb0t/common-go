package commonechomiddleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echomiddleware "github.com/oapi-codegen/echo-middleware"
	oapimiddleware "github.com/psyb0t/common-go/http/oapi-codegen/middleware"
)

// AuthFunc validates the request. Return nil if valid, error if not.
// Only called for endpoints with security requirements in the OpenAPI spec.
// Endpoints with `security: []` bypass this entirely.
type AuthFunc func(ctx context.Context, r *http.Request) error

// CreateDefaultAPIMiddleware creates the API middleware stack with OpenAPI
// validation and panic recovery. If auth is nil, no authentication.
func CreateDefaultAPIMiddleware(
	oas *openapi3.T,
	auth AuthFunc,
) []echo.MiddlewareFunc {
	opts := &echomiddleware.Options{}
	if auth != nil {
		opts.Options.AuthenticationFunc = func(
			ctx context.Context,
			input *openapi3filter.AuthenticationInput,
		) error {
			return auth(ctx, input.RequestValidationInput.Request)
		}
	}

	return []echo.MiddlewareFunc{
		oapimiddleware.OapiValidatorMiddlewareWithOptions(oas, opts),
		middleware.Recover(),
	}
}

// TokenValidator validates a token string. Return nil if valid, error if not.
type TokenValidator func(ctx context.Context, token string) error

// BearerAuth creates an AuthFunc that extracts a Bearer token and validates it.
func BearerAuth(validate TokenValidator) AuthFunc {
	return func(ctx context.Context, r *http.Request) error {
		token := extractBearerToken(r)
		if token == "" {
			return echo.NewHTTPError(
				http.StatusUnauthorized,
				"missing token",
			)
		}

		if err := validate(ctx, token); err != nil {
			return echo.NewHTTPError(
				http.StatusUnauthorized,
				"invalid token",
			)
		}

		return nil
	}
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return ""
	}

	return strings.TrimPrefix(authHeader, prefix)
}
