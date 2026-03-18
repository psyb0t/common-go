package commonhttp

type ErrorResponse struct {
	Error string `json:"error"`
}

type ErrorResponseWithDetails struct {
	Error   string `json:"error"`
	Details any    `json:"details,omitempty"`
}
