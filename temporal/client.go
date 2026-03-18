package commontemporal

import (
	"context"
	"crypto/tls"
	"log/slog"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gonfiguration"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ClientConfig struct {
	HostPort  string `env:"TEMPORAL_CLIENT_HOSTPORT"`
	Namespace string `env:"TEMPORAL_CLIENT_NAMESPACE"`
	APIKey    string `env:"TEMPORAL_CLIENT_APIKEY"`
}

func (c ClientConfig) validate() error {
	if c.HostPort == "" {
		return ErrHostPortRequired
	}

	return nil
}

func parseClientConfig() (ClientConfig, error) {
	cfg := ClientConfig{}

	if err := gonfiguration.Parse(&cfg); err != nil {
		return ClientConfig{}, ctxerrors.Wrap(err, "could not parse config")
	}

	return cfg, nil
}

type Client struct {
	config ClientConfig
	C      client.Client
}

func NewClient(
	ctx context.Context,
	logger *slog.Logger,
) (*Client, error) {
	cfg, err := parseClientConfig()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "could not parse config")
	}

	return NewClientWithConfig(ctx, cfg, logger)
}

func NewClientWithConfig(
	ctx context.Context,
	cfg ClientConfig,
	logger *slog.Logger,
) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, ctxerrors.Wrap(err, "could not validate config")
	}

	clientOptions := createClientOptions(cfg, logger)

	c, err := client.DialContext(ctx, clientOptions)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "failed to create Temporal client")
	}

	return &Client{
		config: cfg,
		C:      c,
	}, nil
}

func (c *Client) Close() {
	c.C.Close()
}

func createClientOptions(
	cfg ClientConfig,
	logger *slog.Logger,
) client.Options {
	opts := client.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
		Logger:    NewSlogAdapter(logger),
	}

	if cfg.APIKey != "" {
		configureAPIKey(&opts, cfg)
	}

	return opts
}

func configureAPIKey(opts *client.Options, cfg ClientConfig) {
	const temporalNamespaceKey = "temporal-namespace"

	opts.ConnectionOptions = client.ConnectionOptions{
		TLS: &tls.Config{}, //nolint:gosec
		DialOptions: []grpc.DialOption{
			grpc.WithUnaryInterceptor(
				func(
					ctx context.Context,
					method string,
					req any,
					reply any,
					cc *grpc.ClientConn,
					invoker grpc.UnaryInvoker,
					callOpts ...grpc.CallOption,
				) error {
					return invoker(
						metadata.AppendToOutgoingContext(
							ctx,
							temporalNamespaceKey,
							cfg.Namespace,
						),
						method,
						req,
						reply,
						cc,
						callOpts...,
					)
				},
			),
		},
	}

	opts.Credentials = client.NewAPIKeyStaticCredentials(cfg.APIKey)
}
