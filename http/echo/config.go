package commonecho

import (
	commonhttp "github.com/psyb0t/common-go/http"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gonfiguration"
)

const envVarNameHTTPEchoListenAddress = "HTTP_ECHO_LISTENADDRESS"

type Config struct {
	ListenAddress string `env:"HTTP_ECHO_LISTENADDRESS"`
	OASPath       string `env:"HTTP_ECHO_OASPATH"`
	SwaggerUIPath string `env:"HTTP_ECHO_SWAGGERUIPATH"`
}

func (c Config) validate() error {
	if c.ListenAddress == "" {
		return ctxerrors.Wrap(ErrListenAddressRequired, "listen address is required")
	}

	return nil
}

func parseConfig() (Config, error) {
	cfg := Config{}

	gonfiguration.SetDefault(
		envVarNameHTTPEchoListenAddress,
		commonhttp.DefaultListenAddress,
	)

	if err := gonfiguration.Parse(&cfg); err != nil {
		return Config{}, ctxerrors.Wrap(err, "could not parse config")
	}

	return cfg, nil
}
