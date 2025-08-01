package claudecode

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// client implements the Client interface
type client struct {
	options *Options
	logger  *slog.Logger
	mu      sync.Mutex
}

// New creates a new Claude client with the given options
func New(opts ...Option) (Client, error) {
	options := DefaultOptions()

	// Apply options
	for _, opt := range opts {
		opt(options)
	}

	// Validate options
	if err := options.validate(); err != nil {
		return nil, err
	}

	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &client{
		options: options,
		logger:  logger.With("component", "claude-client"),
	}, nil
}

// Query sends a single prompt to Claude and blocks until the complete response is received.
// It collects all messages until a ResultMessage is encountered, then returns them as a slice.
// Use this for simple request-response interactions where you need the complete result at once.
func (c *client) Query(ctx context.Context, prompt string, opts ...QueryOption) ([]Message, error) {
	qOpts := &queryOptions{sessionID: "default"}
	for _, opt := range opts {
		opt(qOpts)
	}

	// Create one-shot transport
	transport := NewOneShotTransport(c.options, prompt)

	// Connect
	if err := transport.Connect(ctx); err != nil {
		return nil, err
	}
	defer transport.Close()

	// Receive messages
	msgChan, err := transport.Receive(ctx)
	if err != nil {
		return nil, err
	}

	var messages []Message
	for rawMsg := range msgChan {
		msg, err := ParseMessage(rawMsg)
		if err != nil {
			c.logger.Warn("failed to parse message", "error", err, "data", rawMsg)
			continue
		}
		messages = append(messages, msg)

		// Stop after ResultMessage
		if _, ok := msg.(*ResultMessage); ok {
			break
		}
	}

	return messages, nil
}

// QueryStream sends a query and returns a channel for streaming responses
func (c *client) QueryStream(ctx context.Context, prompt string, opts ...QueryOption) (<-chan Message, error) {
	qOpts := &queryOptions{sessionID: "default"}
	for _, opt := range opts {
		opt(qOpts)
	}

	// Create channel for single prompt
	promptChan := make(chan map[string]any, 1)
	promptChan <- map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": prompt,
		},
		"parent_tool_use_id": nil,
		"session_id":         qOpts.sessionID,
	}
	close(promptChan)

	// Create streaming transport with closeStdinAfterPrompt=true
	transport := NewStreamingTransport(c.options, promptChan, true)

	// Connect
	if err := transport.Connect(ctx); err != nil {
		return nil, err
	}

	// Receive messages
	rawChan, err := transport.Receive(ctx)
	if err != nil {
		transport.Close()
		return nil, err
	}

	// Convert raw messages to typed messages
	msgChan := make(chan Message)

	go func() {
		defer close(msgChan)
		defer transport.Close()

		for rawMsg := range rawChan {
			msg, err := ParseMessage(rawMsg)
			if err != nil {
				c.logger.Warn("failed to parse message", "error", err, "data", rawMsg)
				continue
			}

			select {
			case msgChan <- msg:
			case <-ctx.Done():
				return
			}

			// Stop after ResultMessage
			if _, ok := msg.(*ResultMessage); ok {
				return
			}
		}
	}()

	return msgChan, nil
}

// NewSession creates a new interactive session
func (c *client) NewSession(ctx context.Context, opts ...SessionOption) (Session, error) {
	sOpts := &sessionOptions{}
	for _, opt := range opts {
		opt(sOpts)
	}

	// Create empty prompt channel for interactive mode
	promptChan := make(chan map[string]any)

	// If initial prompt provided, send it
	if sOpts.initialPrompt != "" {
		go func() {
			promptChan <- map[string]any{
				"type": "user",
				"message": map[string]any{
					"role":    "user",
					"content": sOpts.initialPrompt,
				},
				"parent_tool_use_id": nil,
				"session_id":         "default",
			}
		}()
	}

	// Create streaming transport with closeStdinAfterPrompt=false for interactive mode
	transport := NewStreamingTransport(c.options, promptChan, false)

	// Connect
	if err := transport.Connect(ctx); err != nil {
		return nil, err
	}

	return &session{
		transport:  transport,
		logger:     c.logger.With("component", "session"),
		ctx:        ctx,
		promptChan: promptChan,
	}, nil
}

// Close closes the client
func (c *client) Close() error {
	// Currently no persistent resources to clean up
	return nil
}

// session implements the Session interface
type session struct {
	transport  Transport
	logger     *slog.Logger
	ctx        context.Context
	promptChan chan<- map[string]any
	mu         sync.Mutex
	closed     bool
	sessionID  string
}

// Send sends a message in the session
func (s *session) Send(ctx context.Context, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	// Get session ID while we already hold the lock
	sessionID := s.sessionID
	if sessionID == "" {
		sessionID = "default"
	}
	
	msg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": message,
		},
		"parent_tool_use_id": nil,
		"session_id":         sessionID,
	}

	return s.transport.Send(ctx, []map[string]any{msg})
}

// SendMessage sends a pre-constructed message
func (s *session) SendMessage(ctx context.Context, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	// Convert Message to raw format
	// For now, only support UserMessage
	userMsg, ok := msg.(*UserMessage)
	if !ok {
		return fmt.Errorf("%w: only UserMessage supported for sending", ErrInvalidMessage)
	}

	// Get session ID while we already hold the lock
	sessionID := s.sessionID
	if sessionID == "" {
		sessionID = "default"
	}
	
	rawMsg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": userMsg.Content,
		},
		"parent_tool_use_id": nil,
		"session_id":         sessionID,
	}

	return s.transport.Send(ctx, []map[string]any{rawMsg})
}

// Receive returns a channel for receiving messages
func (s *session) Receive(ctx context.Context) (<-chan Message, error) {
	rawChan, err := s.transport.Receive(ctx)
	if err != nil {
		return nil, err
	}

	msgChan := make(chan Message)

	go func() {
		defer close(msgChan)

		for rawMsg := range rawChan {
			msg, err := ParseMessage(rawMsg)
			if err != nil {
				s.logger.Warn("failed to parse message", "error", err, "data", rawMsg)
				continue
			}

			// Update session ID if we get a result message
			if result, ok := msg.(*ResultMessage); ok && result.SessionID != "" {
				s.mu.Lock()
				s.sessionID = result.SessionID
				s.mu.Unlock()
			}

			select {
			case msgChan <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	return msgChan, nil
}

// ReceiveOne receives messages until a ResultMessage is received
func (s *session) ReceiveOne(ctx context.Context) ([]Message, error) {
	msgChan, err := s.Receive(ctx)
	if err != nil {
		return nil, err
	}

	var messages []Message
	for msg := range msgChan {
		messages = append(messages, msg)

		// Stop after ResultMessage
		if _, ok := msg.(*ResultMessage); ok {
			break
		}
	}

	return messages, nil
}

// Interrupt sends an interrupt signal
func (s *session) Interrupt(ctx context.Context) error {
	return s.transport.Interrupt(ctx)
}

// Close closes the session
func (s *session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.promptChan)
	return s.transport.Close()
}

// getSessionID returns the current session ID
func (s *session) getSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sessionID == "" {
		return "default"
	}
	return s.sessionID
}
