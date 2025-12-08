package claudecode

import "context"

// transport defines the internal interface for communication with Claude CLI
type transport interface {
	Connect(ctx context.Context) error
	Close() error
	Send(ctx context.Context, messages []map[string]any) error
	Receive(ctx context.Context) (<-chan map[string]any, error)
	Interrupt(ctx context.Context) error
	IsConnected() bool
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
	// Send sends a user message in the session
	Send(ctx context.Context, message string) error

	// Receive returns a channel for receiving messages
	Receive(ctx context.Context) (<-chan Message, error)

	// ReceiveOne receives messages until a ResultMessage is received
	ReceiveOne(ctx context.Context) ([]Message, error)

	// Interrupt sends an interrupt signal
	Interrupt(ctx context.Context) error

	// Close closes the session
	Close() error
}

// Compile-time interface satisfaction checks
var (
	_ transport = (*subprocessTransport)(nil)
	_ Client    = (*client)(nil)
	_ Session   = (*session)(nil)
)
