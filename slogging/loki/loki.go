package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	commonhttp "github.com/psyb0t/common-go/http"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gonfiguration"
)

const httpTimeout = 5 * time.Second

// ClientConfig holds configuration for the Loki client.
type ClientConfig struct {
	URL string `env:"SLOGGING_LOKI_URL"`
}

func (c ClientConfig) validate() error {
	if c.URL == "" {
		return ctxerrors.New("loki URL is required")
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

// Client pushes log entries directly to Loki's HTTP API.
type Client struct {
	url        string
	httpClient *http.Client
}

// NewClient creates a Loki client with config parsed
// from SLOGGING_LOKI_URL env var.
func NewClient() (*Client, error) {
	cfg, err := parseClientConfig()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "could not parse client config")
	}

	return NewClientWithConfig(cfg)
}

// NewClientWithConfig creates a Loki client from the
// given config.
func NewClientWithConfig(cfg ClientConfig) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, ctxerrors.Wrap(err, "could not validate client config")
	}

	return &Client{
		url: strings.TrimRight(cfg.URL, "/"),
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}, nil
}

type pushPayload struct {
	Streams []stream `json:"streams"`
}

type stream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// Push sends a single log line with labels to Loki.
func (c *Client) Push(
	ctx context.Context,
	labels map[string]string,
	line string,
) {
	ts := strconv.FormatInt(
		time.Now().UnixNano(), 10,
	)

	payload := pushPayload{
		Streams: []stream{
			{
				Stream: labels,
				Values: [][]string{{ts, line}},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	pushURL := c.url + "/loki/api/v1/push"

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, pushURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return
	}

	req.Header.Set(commonhttp.HeaderContentType, commonhttp.ContentTypeJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return
	}

	if err := resp.Body.Close(); err != nil {
		slog.Debug("loki response close failed",
			"error", err,
		)
	}
}

// PushLog formats and sends a log entry directly to
// Loki with the given app name as a label.
func (c *Client) PushLog(
	ctx context.Context,
	appName string,
	source string,
	level string,
	msg string,
	data map[string]any,
) {
	labels := map[string]string{
		"app":    appName,
		"source": source,
		"level":  strings.ToLower(level),
	}

	var line strings.Builder

	line.WriteString(msg)

	for k, v := range data {
		fmt.Fprintf(&line, " %s=%v", k, v)
	}

	c.Push(ctx, labels, line.String())
}

const defaultServiceLabel = "system"

// HandlerConfig holds configuration for the slog Loki handler.
type HandlerConfig struct {
	AppName string `env:"SLOGGING_LOKI_APPNAME"`
	// LabelKeys are slog attribute keys that become Loki
	// labels instead of part of the log line.
	LabelKeys map[string]bool `env:"-"`
}

func (c HandlerConfig) validate() error {
	if c.AppName == "" {
		return ctxerrors.New("loki app name is required")
	}

	return nil
}

func parseHandlerConfig() (HandlerConfig, error) {
	cfg := HandlerConfig{}

	if err := gonfiguration.Parse(&cfg); err != nil {
		return HandlerConfig{}, ctxerrors.Wrap(err, "could not parse config")
	}

	return cfg, nil
}

// Handler implements slog.Handler and pushes log
// records to Loki. Multiple handlers can share one Client.
type Handler struct {
	client    *Client
	appName   string
	labelKeys map[string]bool
	level     slog.Level
	attrs     []slog.Attr
	groups    []string
}

// NewHandler creates a Loki slog handler with config
// parsed from SLOGGING_LOKI_APPNAME env var.
func NewHandler(
	client *Client,
	level slog.Level,
	labelKeys map[string]bool,
) (*Handler, error) {
	cfg, err := parseHandlerConfig()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "could not parse handler config")
	}

	cfg.LabelKeys = labelKeys

	return NewHandlerWithConfig(client, cfg, level)
}

// NewHandlerWithConfig creates a Loki slog handler
// from the given config, using the provided client.
func NewHandlerWithConfig(
	client *Client,
	cfg HandlerConfig,
	level slog.Level,
) (*Handler, error) {
	if err := cfg.validate(); err != nil {
		return nil, ctxerrors.Wrap(err, "could not validate handler config")
	}

	return &Handler{
		client:    client,
		appName:   cfg.AppName,
		labelKeys: cfg.LabelKeys,
		level:     level,
	}, nil
}

func (h *Handler) Enabled(
	_ context.Context,
	level slog.Level,
) bool {
	return level >= h.level
}

func (h *Handler) Handle(
	_ context.Context,
	r slog.Record,
) error {
	labels := map[string]string{
		"app":   h.appName,
		"level": strings.ToLower(r.Level.String()),
	}

	if r.PC != 0 {
		frame, _ := runtime.CallersFrames(
			[]uintptr{r.PC},
		).Next()

		if frame.File != "" {
			labels["source_file"] = frame.File
		}

		if frame.Function != "" {
			labels["source_func"] = frame.Function
		}
	}

	line := &strings.Builder{}
	line.WriteString(r.Message)

	for _, a := range h.attrs {
		h.processAttr(a, labels, line)
	}

	r.Attrs(func(a slog.Attr) bool {
		h.processAttr(a, labels, line)

		return true
	})

	if _, ok := labels["service"]; !ok {
		labels["service"] = defaultServiceLabel
	}

	//nolint:contextcheck // slog handler has no parent ctx
	h.client.Push(context.Background(), labels, line.String())

	return nil
}

func (h *Handler) processAttr(
	a slog.Attr,
	labels map[string]string,
	line *strings.Builder,
) {
	key := a.Key

	for _, g := range h.groups {
		key = g + "." + key
	}

	if h.labelKeys[key] {
		labels[key] = a.Value.String()

		return
	}

	line.WriteString(" ")
	line.WriteString(key)
	line.WriteString("=")
	line.WriteString(a.Value.String())
}

func (h *Handler) WithAttrs(
	attrs []slog.Attr,
) slog.Handler {
	return &Handler{
		client:    h.client,
		appName:   h.appName,
		labelKeys: h.labelKeys,
		level:     h.level,
		attrs:     append(h.attrs, attrs...),
		groups:    h.groups,
	}
}

func (h *Handler) WithGroup(
	name string,
) slog.Handler {
	return &Handler{
		client:    h.client,
		appName:   h.appName,
		labelKeys: h.labelKeys,
		level:     h.level,
		attrs:     h.attrs,
		groups:    append(h.groups, name),
	}
}
