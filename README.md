# Claude Code SDK for Go

Go SDK for [Claude Code](https://docs.anthropic.com/en/docs/claude-code).

## Installation

```bash
go get github.com/mhpenta/claude-code-sdk-go
```

**Prerequisites:**
- Go 1.21+
- Node.js
- Claude Code CLI: `npm install -g @anthropic-ai/claude-code`

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/mhpenta/claude-code-sdk-go/claudecode"
)

func main() {
    client, err := claudecode.New()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    messages, err := client.Query(context.Background(), "What is 2 + 2?")
    if err != nil {
        log.Fatal(err)
    }

    for _, msg := range messages {
        if m, ok := msg.(*claudecode.AssistantMessage); ok {
            for _, block := range m.Content {
                if block.Type == "text" && block.Text != nil {
                    fmt.Println(*block.Text)
                }
            }
        }
    }
}
```

## Usage

### Query with Options

```go
client, err := claudecode.New(
    claudecode.WithSystemPrompt("You are a helpful assistant"),
    claudecode.WithMaxTurns(5),
    claudecode.WithAllowedTools("Read", "Write"),
    claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits),
    claudecode.WithWorkingDirectory("/path/to/project"),
)
```

### Streaming Responses

```go
msgChan, err := client.QueryStream(ctx, "Tell me a story")
if err != nil {
    log.Fatal(err)
}

for msg := range msgChan {
    switch m := msg.(type) {
    case *claudecode.AssistantMessage:
        for _, block := range m.Content {
            if block.Type == "text" && block.Text != nil {
                fmt.Print(*block.Text)
            }
        }
    case *claudecode.ResultMessage:
        fmt.Printf("\nDone: %dms, $%.4f\n", m.DurationMS, *m.TotalCostUSD)
    }
}
```

### Interactive Sessions

```go
session, err := client.NewSession(ctx)
if err != nil {
    log.Fatal(err)
}
defer session.Close()

// Send a message
err = session.Send(ctx, "Let's solve a problem step by step")

// Receive all messages until result
messages, err := session.ReceiveOne(ctx)
```

## API

### Client

```go
type Client interface {
    Query(ctx context.Context, prompt string, opts ...QueryOption) ([]Message, error)
    QueryStream(ctx context.Context, prompt string, opts ...QueryOption) (<-chan Message, error)
    NewSession(ctx context.Context, opts ...SessionOption) (Session, error)
    Close() error
}
```

### Session

```go
type Session interface {
    Send(ctx context.Context, message string) error
    Receive(ctx context.Context) (<-chan Message, error)
    ReceiveOne(ctx context.Context) ([]Message, error)
    Interrupt(ctx context.Context) error
    Close() error
}
```

### Message Types

- `AssistantMessage` - Response from Claude with `Content []ContentBlock`
- `UserMessage` - User input
- `SystemMessage` - System events (tool use notifications, etc.)
- `ResultMessage` - Final result with `DurationMS`, `TotalCostUSD`, `IsError`

### ContentBlock

```go
type ContentBlock struct {
    Type   string      // "text", "tool_use", or "tool_result"
    Text   *string     // For text blocks
    Tool   *ToolUse    // For tool_use blocks
    Result *ToolResult // For tool_result blocks
}
```

## Configuration Options

```go
claudecode.New(
    // Model
    claudecode.WithModel("claude-sonnet-4-20250514"),

    // Prompts
    claudecode.WithSystemPrompt("You are a coding assistant"),
    claudecode.WithAppendSystemPrompt("Always format code properly"),

    // Tools
    claudecode.WithAllowedTools("Read", "Write", "Bash"),
    claudecode.WithDisallowedTools("WebSearch"),
    claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits),

    // Limits
    claudecode.WithMaxTurns(10),
    claudecode.WithMaxThinkingTokens(8000),

    // Context
    claudecode.WithWorkingDirectory("/path/to/project"),
    claudecode.WithAddDirs("./src", "./docs"),

    // Session
    claudecode.WithContinue(),
    claudecode.WithResume("conversation-id"),

    // MCP
    claudecode.WithMCPServer("filesystem", claudecode.MCPServer{
        Type:    claudecode.MCPServerTypeStdio,
        Command: "npx",
        Args:    []string{"@modelcontextprotocol/server-filesystem", "/tmp"},
    }),

    // Other
    claudecode.WithCLIPath("/custom/path/to/claude"),
    claudecode.WithLogger(slog.Default()),
)
```

## Error Handling

```go
import "errors"

messages, err := client.Query(ctx, "Hello")
if err != nil {
    if errors.Is(err, claudecode.ErrNotInstalled) {
        log.Fatal("Claude Code CLI not installed")
    }
    if errors.Is(err, claudecode.ErrConnectionFailed) {
        log.Fatal("Failed to connect")
    }
    log.Fatal(err)
}
```

Sentinel errors: `ErrNotInstalled`, `ErrNotConnected`, `ErrConnectionFailed`, `ErrInvalidMessage`, `ErrStreamClosed`

## Examples

See the [examples](examples/) directory:
- [analyze-sdk](examples/analyze-sdk/) - Code analysis
- [improve-comment](examples/improve-comment/) - Code modification
- [review-readmes](examples/review-readmes/) - Documentation review

## License

MIT
