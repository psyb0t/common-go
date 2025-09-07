package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildURI(t *testing.T) { //nolint:funlen
	testCases := []struct {
		name     string
		scheme   string
		user     string
		pass     string
		host     string
		port     string
		path     string
		expected string
	}{
		{
			name:     "Scheme with host only",
			scheme:   "http",
			user:     "",
			pass:     "",
			host:     "example.com",
			port:     "",
			path:     "",
			expected: "http://example.com",
		},
		{
			name:     "Scheme, user, and host",
			scheme:   "http",
			user:     "guest",
			pass:     "",
			host:     "example.com",
			port:     "",
			path:     "",
			expected: "http://guest@example.com",
		},
		{
			name:     "Scheme, user, pass, and host",
			scheme:   "http",
			user:     "guest",
			pass:     "secret",
			host:     "example.com",
			port:     "",
			path:     "",
			expected: "http://guest:secret@example.com",
		},
		{
			name:     "Scheme, host, and port",
			scheme:   "http",
			user:     "",
			pass:     "",
			host:     "example.com",
			port:     "8080",
			path:     "",
			expected: "http://example.com:8080",
		},
		{
			name:     "Scheme, user, pass, host, and port",
			scheme:   "http",
			user:     "guest",
			pass:     "secret",
			host:     "example.com",
			port:     "8080",
			path:     "",
			expected: "http://guest:secret@example.com:8080",
		},
		{
			name:     "Scheme, user, pass, host, port, and path",
			scheme:   "http",
			user:     "guest",
			pass:     "secret",
			host:     "example.com",
			port:     "8080",
			path:     "/api/v1",
			expected: "http://guest:secret@example.com:8080/api/v1",
		},
		{
			name:     "Scheme with host and path",
			scheme:   "http",
			user:     "",
			pass:     "",
			host:     "example.com",
			port:     "",
			path:     "/api/v1",
			expected: "http://example.com/api/v1",
		},
		{
			name:     "Scheme, user, host, and path",
			scheme:   "http",
			user:     "guest",
			pass:     "",
			host:     "example.com",
			port:     "",
			path:     "/api/v1",
			expected: "http://guest@example.com/api/v1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := BuildURI(
				tc.scheme,
				tc.user,
				tc.pass,
				tc.host,
				tc.port,
				tc.path,
			)

			require.Equal(t, tc.expected, actual)
		})
	}
}
