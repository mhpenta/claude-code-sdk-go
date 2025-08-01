# Claude Code SDK for Go

Go SDK for Claude Code. See the [Claude Code SDK documentation](https://docs.anthropic.com/en/docs/claude-code/sdk) for more information.

## Installation

```bash
go get github.com/mhpenta/claude-code-sdk-go
```

**Prerequisites:**
- Go 1.21+
- Node.js 
- Claude Code: `npm install -g @anthropic-ai/claude-code`

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
        fmt.Printf("%+v\n", msg)
    }
}
```

## Usage

### Basic Query

```go
// Simple query
messages, err := client.Query(ctx, "Hello Claude")
for _, msg := range messages {
    if m, ok := msg.(*claudecode.AssistantMessage); ok {
        for _, block := range m.Content {
            if block.Type == "text" && block.Text != nil {
                fmt.Println(*block.Text)
            }
        }
    }
}

// With options
client, err := claudecode.New(
    claudecode.WithSystemPrompt("You are a helpful assistant"),
    claudecode.WithMaxTurns(1),
)
```

### Streaming Responses

```go
// Stream responses as they arrive
msgChan, err := client.QueryStream(ctx, "Tell me a story")
if err != nil {
    log.Fatal(err)
}

for msg := range msgChan {
    switch m := msg.(type) {
    case *claudecode.AssistantMessage:
        // Process assistant response
    case *claudecode.ResultMessage:
        // Final result with cost and duration
    }
}
```

### Interactive Sessions

```go
// Create an interactive session
session, err := client.NewSession(ctx)
if err != nil {
    log.Fatal(err)
}
defer session.Close()

// Send messages
err = session.Send(ctx, "Let's solve a problem step by step")

// Receive responses
messages, err := session.ReceiveOne(ctx)
```

### Using Tools

```go
client, err := claudecode.New(
    claudecode.WithAllowedTools("Read", "Write", "Bash"),
    claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits),
)

messages, err := client.Query(ctx, "Create a hello.go file")
```

### Working Directory

```go
client, err := claudecode.New(
    claudecode.WithWorkingDirectory("/path/to/project"),
)
```

## API Reference

### Client Interface

```go
type Client interface {
    Query(ctx context.Context, prompt string, opts ...QueryOption) ([]Message, error)
    QueryStream(ctx context.Context, prompt string, opts ...QueryOption) (<-chan Message, error)
    NewSession(ctx context.Context, opts ...SessionOption) (Session, error)
    Close() error
}
```

### Session Interface

```go
type Session interface {
    Send(ctx context.Context, message string) error
    Receive(ctx context.Context) (<-chan Message, error)
    ReceiveOne(ctx context.Context) ([]Message, error)
    Interrupt(ctx context.Context) error
    Close() error
}
```

### Types

See [claudecode/message.go](claudecode/message.go) for complete type definitions:
- `Options` - Configuration options
- `AssistantMessage`, `UserMessage`, `SystemMessage`, `ResultMessage` - Message types
- `TextBlock`, `ToolUse`, `ToolResult` - Content blocks

## Error Handling

```go
import "errors"

// Sentinel errors
var (
    ErrClaudeNotInstalled = errors.New("claude-code: CLI not installed")
    ErrNotConnected       = errors.New("claude-code: not connected")
    ErrConnectionFailed   = errors.New("claude-code: connection failed")
)

// Error handling
messages, err := client.Query(ctx, "Hello")
if err != nil {
    if errors.Is(err, claudecode.ErrClaudeNotInstalled) {
        log.Fatal("Please install Claude Code")
    }
    if errors.Is(err, claudecode.ErrConnectionFailed) {
        log.Fatal("Failed to connect to Claude")
    }
}

// Process errors include exit codes
if procErr, ok := err.(*claudecode.ProcessError); ok {
    log.Printf("Process failed with exit code: %d", procErr.ExitCode)
}
```

See [claudecode/errors.go](claudecode/errors.go) for all error types.

## Configuration Options

```go
client, err := claudecode.New(
    // Model selection
    claudecode.WithModel("claude-3-opus-20240229"),
    
    // System prompts
    claudecode.WithSystemPrompt("You are a coding assistant"),
    
    // Tool permissions
    claudecode.WithAllowedTools("Read", "Write"),
    claudecode.WithPermissionMode(claudecode.PermissionModeDefault),
    
    // Conversation limits
    claudecode.WithMaxTurns(10),
    
    // Working directory
    claudecode.WithWorkingDirectory("/path/to/project"),
    
    // Logging
    claudecode.WithLogger(slog.Default()),
)
```

## Available Tools

See the [Claude Code documentation](https://docs.anthropic.com/en/docs/claude-code/settings#tools-available-to-claude) for a complete list of available tools.

## Examples

- [Simple Usage](examples/simple/) - Basic queries and streaming
- [Code Analysis](examples/analyze-sdk/) - Analyze code architecture
- [Code Modification](examples/improve-comment/) - Improve code documentation
- [Documentation Review](examples/review-readmes/) - Review and update READMEs

See the [examples directory](examples/) for more complete examples.

## License

MIT