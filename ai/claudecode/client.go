package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	commonhttp "github.com/psyb0t/common-go/http"
	"github.com/psyb0t/ctxerrors"
)

const (
	defaultTimeout = 5 * time.Minute
	defaultModel   = "haiku"
	logComponent   = "claudecode-client"
)

// Runner is the interface for interacting with a Claude Code instance.
type Runner interface {
	Run(ctx context.Context, req RunRequest) (*RunResult, error)
	UploadFile(
		ctx context.Context, filePath string, content []byte,
	) (*FileInfo, error)
	DownloadFile(
		ctx context.Context, filePath string,
	) ([]byte, error)
	DeleteFile(ctx context.Context, filePath string) error
	Health(ctx context.Context) error
}

// Client implements Runner by calling Claude Code's HTTP API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithToken sets the auth token.
func WithToken(token string) Option {
	return func(c *Client) { c.token = token }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// New creates a Claude Code API client.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	slog.Debug("claudecode client created",
		"component", logComponent,
		"baseURL", c.baseURL,
	)

	return c
}

type RunRequest struct {
	Prompt             string `json:"prompt"`
	Workspace          string `json:"workspace,omitempty"`
	Model              string `json:"model,omitempty"`
	SystemPrompt       string `json:"systemPrompt,omitempty"`
	AppendSystemPrompt string `json:"appendSystemPrompt,omitempty"`
	JSONSchema         string `json:"jsonSchema,omitempty"`
	Effort             string `json:"effort,omitempty"`
	OutputFormat       string `json:"outputFormat,omitempty"`
	NoContinue         bool   `json:"noContinue,omitempty"`
	Resume             string `json:"resume,omitempty"`
	FireAndForget      bool   `json:"fireAndForget,omitempty"`
}

// Usage holds token usage for a single model call.
type Usage struct {
	InputTokens         int `json:"inputTokens"`
	OutputTokens        int `json:"outputTokens"`
	CacheCreationTokens int `json:"cacheCreationInputTokens,omitempty"`
	CacheReadTokens     int `json:"cacheReadInputTokens,omitempty"`
}

// ModelUsage tracks per-model token usage.
//
//nolint:tagliatelle // external API uses non-standard camelCase
type ModelUsage struct {
	InputTokens         int     `json:"inputTokens"`
	OutputTokens        int     `json:"outputTokens"`
	CacheReadTokens     int     `json:"cacheReadInputTokens,omitempty"`
	CacheCreationTokens int     `json:"cacheCreationInputTokens,omitempty"`
	CostUSD             float64 `json:"costUSD,omitempty"`
	ContextWindow       int     `json:"contextWindow,omitempty"`
	MaxOutputTokens     int     `json:"maxOutputTokens,omitempty"`
}

// Iteration is one assistant turn with tool calls and text.
type Iteration struct {
	TurnNumber int             `json:"turnNumber"`
	Content    json.RawMessage `json:"content"`
	ToolUses   []ToolUse       `json:"toolUses,omitempty"`
	Usage      *Usage          `json:"usage,omitempty"`
}

// ToolUse represents a single tool invocation within an iteration.
type ToolUse struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// VerboseTurn is a single conversation turn from json-verbose output.
type VerboseTurn struct {
	Role    string            `json:"role"`
	Content []json.RawMessage `json:"content"`
}

// VerboseSystem holds init metadata from json-verbose output.
type VerboseSystem struct {
	SessionID string   `json:"sessionId"`
	Model     string   `json:"model"`
	Cwd       string   `json:"cwd"`
	Tools     []string `json:"tools"`
}

type RunResult struct {
	Type              string                `json:"type"`
	Subtype           string                `json:"subtype,omitempty"`
	IsError           bool                  `json:"isError"`
	Result            string                `json:"result"`
	NumTurns          int                   `json:"numTurns"`
	DurationMS        int                   `json:"durationMs"`
	TotalCost         float64               `json:"totalCostUsd"`
	SessionID         string                `json:"sessionId"`
	Usage             *Usage                `json:"usage,omitempty"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage,omitempty"`
	PermissionDenials []string              `json:"permissionDenials,omitempty"`
	Iterations        []Iteration           `json:"iterations,omitempty"`
	Turns             []VerboseTurn         `json:"turns,omitempty"`
	System            *VerboseSystem        `json:"system,omitempty"`
}

// FileInfo is the response from file upload operations.
type FileInfo struct {
	Status string `json:"status"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
}

//nolint:funlen // debug logging adds lines
func (c *Client) Run(
	ctx context.Context,
	req RunRequest,
) (*RunResult, error) {
	if req.Model == "" {
		req.Model = defaultModel
	}

	slog.Debug("claude run starting",
		"component", logComponent,
		"model", req.Model,
		"workspace", req.Workspace,
		"promptLength", len(req.Prompt),
	)

	start := time.Now()

	body, err := json.Marshal(req)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "marshal run request")
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.baseURL+"/run",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "create run request")
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		slog.Error("claude run HTTP failed",
			"component", logComponent,
			"error", err,
		)

		return nil, ctxerrors.Wrap(err, "execute run request")
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)

		slog.Error("claude run non-200 response",
			"component", logComponent,
			"status", resp.StatusCode,
			"body", string(errBody),
		)

		return nil, ctxerrors.New(fmt.Sprintf(
			"run failed: HTTP %d: %s",
			resp.StatusCode, string(errBody),
		))
	}

	var result RunResult
	if err := json.NewDecoder(resp.Body).Decode(
		&result,
	); err != nil {
		return nil, ctxerrors.Wrap(err, "decode run result")
	}

	slog.Info("claude run completed",
		"component", logComponent,
		"model", req.Model,
		"workspace", req.Workspace,
		"isError", result.IsError,
		"numTurns", result.NumTurns,
		"durationMs", result.DurationMS,
		"totalCostUsd", result.TotalCost,
		"clientDuration", time.Since(start).Round(time.Millisecond),
	)

	return &result, nil
}

func (c *Client) UploadFile(
	ctx context.Context,
	filePath string,
	content []byte,
) (*FileInfo, error) {
	slog.Debug("uploading file",
		"component", logComponent,
		"path", filePath,
		"size", len(content),
	)

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut,
		c.filesURL(filePath),
		bytes.NewReader(content),
	)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "create upload request")
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "upload file")
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		slog.Error("upload file failed",
			"component", logComponent,
			"path", filePath,
			"status", resp.StatusCode,
		)

		return nil, ctxerrors.New(fmt.Sprintf(
			"upload failed: HTTP %d", resp.StatusCode,
		))
	}

	var info FileInfo
	if err := json.NewDecoder(resp.Body).Decode(
		&info,
	); err != nil {
		return nil, ctxerrors.Wrap(
			err, "decode upload response",
		)
	}

	slog.Debug("file uploaded",
		"component", logComponent,
		"path", filePath,
		"serverPath", info.Path,
		"size", info.Size,
	)

	return &info, nil
}

//nolint:funlen // debug logging adds lines
func (c *Client) DownloadFile(
	ctx context.Context,
	filePath string,
) ([]byte, error) {
	slog.Debug("downloading file",
		"component", logComponent,
		"path", filePath,
	)

	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet,
		c.filesURL(filePath), nil,
	)
	if err != nil {
		return nil, ctxerrors.Wrap(
			err, "create download request",
		)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "download file")
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		slog.Warn("file not found",
			"component", logComponent,
			"path", filePath,
		)

		return nil, ctxerrors.New(
			"file not found: " + filePath,
		)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("download file failed",
			"component", logComponent,
			"path", filePath,
			"status", resp.StatusCode,
		)

		return nil, ctxerrors.New(fmt.Sprintf(
			"download failed: HTTP %d", resp.StatusCode,
		))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "read file body")
	}

	slog.Debug("file downloaded",
		"component", logComponent,
		"path", filePath,
		"size", len(data),
	)

	return data, nil
}

func (c *Client) DeleteFile(
	ctx context.Context,
	filePath string,
) error {
	slog.Debug("deleting file",
		"component", logComponent,
		"path", filePath,
	)

	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete,
		c.filesURL(filePath), nil,
	)
	if err != nil {
		return ctxerrors.Wrap(err, "create delete request")
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ctxerrors.Wrap(err, "delete file")
	}

	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("delete file failed",
			"component", logComponent,
			"path", filePath,
			"status", resp.StatusCode,
		)

		return ctxerrors.New(fmt.Sprintf(
			"delete failed: HTTP %d", resp.StatusCode,
		))
	}

	slog.Debug("file deleted",
		"component", logComponent,
		"path", filePath,
	)

	return nil
}

// Health checks if the Claude Code instance is responsive.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet,
		c.baseURL+"/health", nil,
	)
	if err != nil {
		return ctxerrors.Wrap(err, "create health request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ctxerrors.Wrap(err, "health check")
	}

	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ctxerrors.New(fmt.Sprintf(
			"health check failed: HTTP %d",
			resp.StatusCode,
		))
	}

	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set(commonhttp.HeaderContentType, commonhttp.ContentTypeJSON)

	if c.token != "" {
		req.Header.Set(
			commonhttp.HeaderAuthorization,
			commonhttp.AuthSchemeBearer+c.token,
		)
	}
}

func (c *Client) filesURL(path string) string {
	path = strings.TrimLeft(path, "/")

	return c.baseURL + "/files/" + path
}
