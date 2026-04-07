# claudecode

Go client for the [docker-claude-code](https://github.com/psyb0t/docker-claude-code) HTTP API.

## Usage

```go
import "github.com/psyb0t/common-go/ai/claudecode"

client := claudecode.New("http://localhost:8080",
    claudecode.WithToken("my-api-token"),
    claudecode.WithTimeout(10*time.Minute),
)

result, err := client.Run(ctx, claudecode.RunRequest{
    Prompt:    "refactor the auth middleware",
    Workspace: "my-project",
    Model:     "sonnet",
})
```

## Runner Interface

```go
type Runner interface {
    Run(ctx context.Context, req RunRequest) (*RunResult, error)
    UploadFile(ctx context.Context, filePath string, content []byte) (*FileInfo, error)
    DownloadFile(ctx context.Context, filePath string) ([]byte, error)
    DeleteFile(ctx context.Context, filePath string) error
    Health(ctx context.Context) error
}
```

## RunRequest

| Field              | JSON                 | Description                                        |
| ------------------ | -------------------- | -------------------------------------------------- |
| Prompt             | `prompt`             | The prompt to send                                 |
| Workspace          | `workspace`          | Subdirectory under `/workspaces`                   |
| Model              | `model`              | Model name (default: `haiku`)                      |
| SystemPrompt       | `systemPrompt`       | Override the system prompt                         |
| AppendSystemPrompt | `appendSystemPrompt` | Append to the system prompt                        |
| JSONSchema         | `jsonSchema`         | JSON schema for structured output                  |
| Effort             | `effort`             | Effort level                                       |
| OutputFormat       | `outputFormat`       | `"json"` (default) or `"json-verbose"` (see below) |
| NoContinue         | `noContinue`         | Don't use `--continue` flag                        |
| Resume             | `resume`             | Resume a previous session by ID                    |
| FireAndForget      | `fireAndForget`      | Return immediately without waiting for result      |

## RunResult

Response from a completed run:

- `Result` — the final text output
- `SessionID` — session ID for resuming
- `IsError` — whether the run failed
- `NumTurns`, `DurationMS`, `TotalCost` — execution stats
- `Usage` / `ModelUsage` — token usage breakdown
- `Iterations` — per-turn content and tool calls
- `Turns` — full conversation history (only with `json-verbose`)
- `System` — init metadata: session ID, model, cwd, tools (only with `json-verbose`)

## json-verbose Output Format

Set `OutputFormat: "json-verbose"` to get the full conversation history including every tool call and tool result in the response.

```go
result, err := client.Run(ctx, claudecode.RunRequest{
    Prompt:       "check if the tests pass",
    Workspace:    "my-project",
    OutputFormat: "json-verbose",
})

// result.Turns contains the full conversation:
// - {Role: "assistant", Content: [...text blocks, tool_use blocks...]}
// - {Role: "tool_result", Content: [...tool results with content and is_error...]}
//
// result.System contains init metadata:
// - SessionID, Model, Cwd, Tools
```

## File Operations

```go
// Upload a file into the workspace
info, err := client.UploadFile(ctx, "my-project/config.yaml", data)

// Download a file from the workspace
data, err := client.DownloadFile(ctx, "my-project/output.json")

// Delete a file
err := client.DeleteFile(ctx, "my-project/tmp.txt")
```

## Health Check

```go
err := client.Health(ctx)
```

## Mocking

A `Runner` mock is generated via `go generate` for use in tests:

```go
import "github.com/psyb0t/common-go/ai/claudecode/mocks"

mock := mocks.NewMockRunner(t)
```
