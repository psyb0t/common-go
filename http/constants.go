package commonhttp

const (
	DefaultListenAddress = "0.0.0.0:8080"
	DefaultAPIRootPath   = "/api"

	StandardAPIOASPath       = "/openapi.yaml"
	StandardAPISwaggerUIPath = "/docs/*"

	ContentTypeYAML            = "application/x-yaml"
	ContentTypeJSON            = "application/json"
	ContentTypeFormURLEncoded  = "application/x-www-form-urlencoded"
	ContentTypeTextEventStream = "text/event-stream"

	HeaderContentType   = "Content-Type"
	HeaderAuthorization = "Authorization"
	HeaderUserAgent     = "User-Agent"

	AuthSchemeBearer = "Bearer "
)
