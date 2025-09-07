package utils

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/psyb0t/ctxerrors"
)

// BuildURI constructs a URI based on the provided parameters.
// - scheme: "http", "https", "amqp", "amqps", etc.
// - user: username (optional)
// - pass: password (optional)
// - host: host of the server (e.g., "localhost")
// - port: port (optional)
// - path: additional path or endpoint (optional)
func BuildURI(scheme, user, pass, host, port, path string) string {
	// Initialize the URI string
	uri := scheme + "://"

	// Handle user and password
	if user != "" {
		uriPart := user
		if pass != "" {
			uriPart = url.UserPassword(user, pass).String()
		}

		uri += uriPart + "@"
	}

	// Append host
	uri += host

	// Append port if provided
	if port != "" {
		uri += ":" + port
	}

	// Append path if provided
	if path != "" {
		uri += path
	}

	return uri
}

func JoinURLPaths(baseURL string, paths ...string) string {
	// Ensure we're working with clean paths
	cleanPaths := make([]string, 0, len(paths))

	for _, path := range paths {
		// Trim leading and trailing slashes
		path = strings.Trim(path, "/")
		if path != "" {
			cleanPaths = append(cleanPaths, path)
		}
	}

	// Trim trailing slash from baseURL if present
	baseURL = strings.TrimRight(baseURL, "/")

	// If there are no paths to join, just return the baseURL
	if len(cleanPaths) == 0 {
		return baseURL
	}

	// Join the paths with the proper separator
	joinedPath := strings.Join(cleanPaths, "/")

	// Return the complete URL
	return baseURL + "/" + joinedPath
}

func GetURLWithParams(baseURL string, params map[string]any) (string, error) {
	if len(params) == 0 {
		return baseURL, nil
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", ctxerrors.Wrap(err, "failed to parse base URL")
	}

	query := parsedURL.Query()
	for k, v := range params {
		query.Set(k, fmt.Sprintf("%v", v))
	}

	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}
