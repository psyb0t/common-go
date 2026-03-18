package urlutil

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/psyb0t/ctxerrors"
)

// BuildURI constructs a URI based on the provided parameters.
func BuildURI(scheme, user, pass, host, port, path string) string {
	uri := scheme + "://"

	if user != "" {
		uriPart := user
		if pass != "" {
			uriPart = url.UserPassword(user, pass).String()
		}

		uri += uriPart + "@"
	}

	uri += host

	if port != "" {
		uri += ":" + port
	}

	if path != "" {
		uri += path
	}

	return uri
}

func JoinURLPaths(baseURL string, paths ...string) string {
	cleanPaths := make([]string, 0, len(paths))

	for _, path := range paths {
		path = strings.Trim(path, "/")
		if path != "" {
			cleanPaths = append(cleanPaths, path)
		}
	}

	baseURL = strings.TrimRight(baseURL, "/")

	if len(cleanPaths) == 0 {
		return baseURL
	}

	return baseURL + "/" + strings.Join(cleanPaths, "/")
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
