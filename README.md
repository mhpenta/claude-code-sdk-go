# Claude Code SDK for Go

A Go SDK for interacting with Claude Code, providing programmatic access to Claude's capabilities through a clean, idiomatic Go interface.

## Features

- 🚀 **Simple API**: Easy-to-use interfaces for one-shot queries and interactive sessions
- 🔄 **Streaming Support**: Real-time message streaming for responsive applications
- 🎯 **Type Safety**: Fully typed messages and options
- 🛡️ **Error Handling**: Comprehensive error types with sentinel errors
- 🔧 **Flexible Configuration**: Extensive options for customizing behavior
- 📝 **Context Support**: Full `context.Context` integration for cancellation and timeouts

## Installation

```bash
go get github.com/mhpenta/claude-code-sdk-go
```

### Prerequisites

- Go 1.21 or later
- Node.js installed
- Claude Code CLI installed: `npm install -g @anthropic-ai/claude-code`

## Quick Start

### Simple Query

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/mhpenta/claude-code-sdk-go/claudecode"
)

func main() {
    // Create client
    client, err := claudecode.New()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Send query
    messages, err := client.Query(context.Background(), "What is 2 + 2?")
    if err != nil {
        log.Fatal(err)
    }
    
    // Process response
    for _, msg := range messages {
        if m, ok := msg.(*claude.AssistantMessage); ok {
            for _, block := range m.Content {
                if block.Type == "text" && block.Text != nil {
                    fmt.Printf("Claude: %s\n", *block.Text)
                }
            }
        }
    }
}
```

### Streaming Responses

```go
// Stream responses as they arrive
msgChan, err := client.QueryStream(context.Background(), "Tell me a story")
if err != nil {
    log.Fatal(err)
}

for msg := range msgChan {
    // Process each message as it arrives
    fmt.Printf("Received: %T\n", msg)
}
```

### Interactive Session

```go
// Create an interactive session
session, err := client.NewSession(context.Background())
if err != nil {
    log.Fatal(err)
}
defer session.Close()

// Send message
err = session.Send(context.Background(), "Let's play 20 questions")
if err != nil {
    log.Fatal(err)
}

// Receive response
messages, err := session.ReceiveOne(context.Background())
if err != nil {
    log.Fatal(err)
}
```

## Configuration

### Client Options

```go
client, err := claude.New(
    claude.WithModel("claude-3-opus-20240229"),
    claude.WithSystemPrompt("You are a helpful assistant"),
    claude.WithMaxTurns(10),
    claude.WithPermissionMode(claude.PermissionModeAcceptEdits),
    claude.WithWorkingDirectory("/path/to/project"),
    claude.WithLogger(slog.Default()),
)
```

### Available Options

- `WithModel(model)` - Set the Claude model to use
- `WithSystemPrompt(prompt)` - Set system prompt
- `WithMaxTurns(n)` - Limit conversation turns
- `WithPermissionMode(mode)` - Control tool permissions
- `WithWorkingDirectory(dir)` - Set working directory
- `WithAllowedTools(tools...)` - Specify allowed tools
- `WithLogger(logger)` - Set custom logger
- `WithCLIPath(path)` - Override Claude CLI path

## Message Types

The SDK provides typed message structures:

- `UserMessage` - Messages from the user
- `AssistantMessage` - Claude's responses with content blocks
- `SystemMessage` - System events and metadata
- `ResultMessage` - Conversation summary with cost and usage

## Error Handling

The SDK provides comprehensive error handling with sentinel errors:

```go
errors.Is(err, claude.ErrClaudeNotInstalled)  // CLI not found
errors.Is(err, claude.ErrNotConnected)         // Not connected
errors.Is(err, claude.ErrConnectionFailed)     // Connection failed
errors.Is(err, claude.ErrInvalidMessage)       // Invalid message
```

## Architecture

The SDK follows a clean architecture with clear separation of concerns:

- **Client Interface**: High-level API for users
- **Transport Layer**: Handles communication (subprocess, future: HTTP/gRPC)
- **Message Types**: Type-safe message handling
- **Options**: Flexible configuration system

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

This SDK is inspired by the official Python SDK and follows Go best practices for API design.