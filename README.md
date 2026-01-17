# Claude Code SDK for Go

Go SDK for [Claude Code](https://code.claude.com/docs/en/overview).

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

// Dynamic session control
err = session.SetPermissionMode(ctx, claudecode.PermissionModeAcceptEdits)
err = session.SetModel(ctx, "claude-sonnet-4-20250514")

// Get session ID
fmt.Println("Session ID:", session.SessionID())
```

### File Checkpointing (Rewind Support)

```go
client, err := claudecode.New(
    claudecode.WithEnableFileCheckpointing(),
)

session, err := client.NewSession(ctx)
// ... make changes ...

// Rewind files to a previous checkpoint
err = session.RewindFiles(ctx, userMessageID)
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
    SetPermissionMode(ctx context.Context, mode PermissionMode) error
    SetModel(ctx context.Context, model string) error
    RewindFiles(ctx context.Context, userMessageID string) error
    GetServerInfo(ctx context.Context) (map[string]any, error)
    SessionID() string
    Close() error
}
```

### Message Types

- `AssistantMessage` - Response from Claude with `Content []ContentBlock`, `Model`, `Error`
- `UserMessage` - User input with `Content`, `UUID`, `ParentToolUseID`
- `SystemMessage` - System events (tool use notifications, etc.)
- `ResultMessage` - Final result with `DurationMS`, `TotalCostUSD`, `IsError`, `StructuredOutput`
- `StreamEvent` - Partial message updates during streaming (with `UUID`, `SessionID`, `Event`)

### ContentBlock

```go
type ContentBlock struct {
    Type     string        // "text", "tool_use", "tool_result", or "thinking"
    Text     *string       // For text blocks
    Tool     *ToolUse      // For tool_use blocks
    Result   *ToolResult   // For tool_result blocks
    Thinking *ThinkingBlock // For thinking blocks (extended thinking)
}

type ThinkingBlock struct {
    Thinking  string // The thinking content
    Signature string // Signature for verification
}
```

## Configuration Options

### Basic Options

```go
claudecode.New(
    // Model
    claudecode.WithModel("claude-sonnet-4-20250514"),
    claudecode.WithFallbackModel("claude-haiku-3-20240307"),

    // Prompts
    claudecode.WithSystemPrompt("You are a coding assistant"),
    claudecode.WithSystemPromptPreset(claudecode.SystemPromptPreset{
        Type:   "preset",
        Preset: "claude_code",
        Append: "Additional instructions",
    }),

    // Tools
    claudecode.WithTools("Read", "Write", "Bash"), // Explicit tool list
    claudecode.WithTools(), // Empty list disables all built-in tools
    claudecode.WithToolsPreset(claudecode.ToolsPreset{
        Type:   "preset",
        Preset: "claude_code",
    }),
    claudecode.WithAllowedTools("Read", "Write", "Bash"),
    claudecode.WithDisallowedTools("WebSearch"),

    // Permission Modes
    claudecode.WithPermissionMode(claudecode.PermissionModeDefault),       // Prompts for dangerous tools
    claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits),   // Auto-accepts file edits
    claudecode.WithPermissionMode(claudecode.PermissionModePlan),          // Plan-only mode
    claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions), // Allow all (use with caution)

    // Limits
    claudecode.WithMaxTurns(10),
    claudecode.WithMaxThinkingTokens(8000),
    claudecode.WithMaxBudgetUSD(1.00), // Spending limit

    // Context
    claudecode.WithWorkingDirectory("/path/to/project"),
    claudecode.WithAddDirs("./src", "./docs"),

    // Session Management
    claudecode.WithContinue(),
    claudecode.WithResume("conversation-id"),
    claudecode.WithForkSession(), // Fork instead of continuing when resuming
)
```

### Advanced Options

```go
claudecode.New(
    // Beta Features
    claudecode.WithBetas(claudecode.SdkBetaContext1M), // Extended context window

    // Setting Sources
    claudecode.WithSettingSources(
        claudecode.SettingSourceUser,
        claudecode.SettingSourceProject,
        claudecode.SettingSourceLocal,
    ),

    // Environment Variables
    claudecode.WithEnv(map[string]string{
        "CUSTOM_VAR": "value",
    }),

    // Extra CLI Arguments
    claudecode.WithExtraArg("verbose", nil),           // --verbose
    claudecode.WithExtraArg("timeout", ptr("30000")),  // --timeout 30000

    // Structured Output
    claudecode.WithOutputFormat(claudecode.OutputFormat{
        Type: "json_schema",
        Schema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "result": map[string]any{"type": "string"},
            },
        },
    }),

    // File Checkpointing
    claudecode.WithEnableFileCheckpointing(),

    // Partial Message Streaming
    claudecode.WithIncludePartialMessages(),

    // Other
    claudecode.WithCLIPath("/custom/path/to/claude"),
    claudecode.WithLogger(slog.Default()),
)
```

### MCP Server Configuration

```go
claudecode.New(
    claudecode.WithMCPServer("filesystem", claudecode.MCPServer{
        Type:    claudecode.MCPServerTypeStdio,
        Command: "npx",
        Args:    []string{"@modelcontextprotocol/server-filesystem", "/tmp"},
        Env:     map[string]string{"DEBUG": "true"},
    }),
    claudecode.WithMCPServer("api", claudecode.MCPServer{
        Type:    claudecode.MCPServerTypeHTTP,
        URL:     "https://api.example.com/mcp",
        Headers: map[string]string{"Authorization": "Bearer token"},
    }),
)
```

### Custom Agents

```go
claudecode.New(
    claudecode.WithAgent("code-reviewer", claudecode.AgentDefinition{
        Description: "Reviews code for best practices",
        Prompt:      "You are a code reviewer. Focus on...",
        Tools:       []string{"Read", "Grep"},
        Model:       claudecode.AgentModelSonnet,
    }),
)
```

### Sandbox Configuration

```go
claudecode.New(
    claudecode.WithSandbox(claudecode.SandboxSettings{
        Enabled:                  true,
        AutoAllowBashIfSandboxed: true,
        ExcludedCommands:         []string{"git", "docker"},
        Network: &claudecode.SandboxNetworkConfig{
            AllowUnixSockets:  []string{"/var/run/docker.sock"},
            AllowLocalBinding: true,
        },
    }),
)
```

### Plugin Configuration

```go
claudecode.New(
    claudecode.WithPlugin(claudecode.PluginConfig{
        Type: claudecode.PluginTypeLocal,
        Path: "/path/to/plugin",
    }),
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

// Check for assistant message errors
for _, msg := range messages {
    if m, ok := msg.(*claudecode.AssistantMessage); ok {
        if m.Error != "" {
            switch m.Error {
            case claudecode.AssistantMessageErrorRateLimit:
                log.Println("Rate limited")
            case claudecode.AssistantMessageErrorBillingError:
                log.Println("Billing error")
            }
        }
    }
}
```

Sentinel errors: `ErrNotInstalled`, `ErrNotConnected`, `ErrConnectionFailed`, `ErrInvalidMessage`, `ErrStreamClosed`

## Examples

See the [examples](examples/) directory:
- [analyze-sdk](examples/analyze-sdk/) - Code analysis
- [improve-comment](examples/improve-comment/) - Code modification
- [review-readmes](examples/review-readmes/) - Documentation review
- [test-exit-handling](examples/test-exit-handling/) - Exit handling validation

## License

MIT
