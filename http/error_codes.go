package commonhttp

import "net/http"

const (
	// 4xx Client Errors
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeGone                = "GONE"
	ErrCodeUnprocessableEntity = "UNPROCESSABLE_ENTITY"
	ErrCodeTooManyRequests     = "TOO_MANY_REQUESTS"
	ErrCodeValidation          = "VALIDATION"
	ErrCodeRateLimited         = "RATE_LIMITED"

	// 5xx Server Errors
	ErrCodeInternalServer     = "INTERNAL_SERVER"
	ErrCodeNotImplemented     = "NOT_IMPLEMENTED"
	ErrCodeBadGateway         = "BAD_GATEWAY"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeGatewayTimeout     = "GATEWAY_TIMEOUT"
)

// ErrCodeFromHTTPStatus maps an HTTP status code to an error code string.
func ErrCodeFromHTTPStatus(status int) string {
	m := map[int]string{
		http.StatusBadRequest:          ErrCodeBadRequest,
		http.StatusUnauthorized:        ErrCodeUnauthorized,
		http.StatusForbidden:           ErrCodeForbidden,
		http.StatusNotFound:            ErrCodeNotFound,
		http.StatusMethodNotAllowed:    ErrCodeMethodNotAllowed,
		http.StatusConflict:            ErrCodeConflict,
		http.StatusGone:                ErrCodeGone,
		http.StatusUnprocessableEntity: ErrCodeUnprocessableEntity,
		http.StatusTooManyRequests:     ErrCodeTooManyRequests,
		http.StatusInternalServerError: ErrCodeInternalServer,
		http.StatusNotImplemented:      ErrCodeNotImplemented,
		http.StatusBadGateway:          ErrCodeBadGateway,
		http.StatusServiceUnavailable:  ErrCodeServiceUnavailable,
		http.StatusGatewayTimeout:      ErrCodeGatewayTimeout,
	}

	code, ok := m[status]
	if !ok {
		return ErrCodeInternalServer
	}

	return code
}
