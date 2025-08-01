package claudecode

import (
	"context"
	"io"
)

// Transport defines the interface for communication with Claude
type Transport interface {
	// Connect establishes a connection
	Connect(ctx context.Context) error

	// Close terminates the connection
	Close() error

	// Send sends messages to Claude
	Send(ctx context.Context, messages []map[string]any) error

	// Receive returns a channel for receiving messages
	Receive(ctx context.Context) (<-chan map[string]any, error)

	// Interrupt sends an interrupt signal
	Interrupt(ctx context.Context) error

	// IsConnected returns true if the transport is connected
	IsConnected() bool
}

// StreamingTransport extends Transport with streaming capabilities
type StreamingTransport interface {
	Transport

	// SendStream sends a stream of messages
	SendStream(ctx context.Context, messages <-chan map[string]any) error
}

// Client is the main interface for interacting with Claude
type Client interface {
	// Query sends a one-shot query and returns all messages
	Query(ctx context.Context, prompt string, opts ...QueryOption) ([]Message, error)

	// QueryStream sends a query and returns a channel for streaming responses
	QueryStream(ctx context.Context, prompt string, opts ...QueryOption) (<-chan Message, error)

	// NewSession creates a new interactive session
	NewSession(ctx context.Context, opts ...SessionOption) (Session, error)

	// Close closes the client and releases resources
	Close() error
}

// Session represents an interactive conversation session
type Session interface {
	// Send sends a message in the session
	Send(ctx context.Context, message string) error

	// SendMessage sends a pre-constructed message
	SendMessage(ctx context.Context, msg Message) error

	// Receive returns a channel for receiving messages
	Receive(ctx context.Context) (<-chan Message, error)

	// ReceiveOne receives messages until a ResultMessage is received
	ReceiveOne(ctx context.Context) ([]Message, error)

	// Interrupt sends an interrupt signal
	Interrupt(ctx context.Context) error

	// Close closes the session
	Close() error
}

// Ensure interfaces implement io.Closer where appropriate
var (
	_ io.Closer = (Transport)(nil)
	_ io.Closer = (Client)(nil)
	_ io.Closer = (Session)(nil)
)
