package httputil

import "errors"

// ErrUnexpectedHTTPStatusCode is returned when an HTTP response carries a status
// code the caller did not expect. It lives here, in the HTTP utility package,
// rather than in the cross-project sentinel vocabulary — an HTTP-status error is
// HTTP-specific, not a general-purpose one.
var ErrUnexpectedHTTPStatusCode = errors.New("unexpected http status code")
