package claudecode

import (
	"context"
	"log/slog"
	"sync"
)

// client implements the Client interface
type client struct {
	options *Options
	logger  *slog.Logger
}

// New creates a new Claude client with the given options
func New(opts ...Option) (Client, error) {
	options := DefaultOptions()

	for _, opt := range opts {
		opt(options)
	}

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

	transport := newOneShotTransport(c.options, prompt)

	if err := transport.Connect(ctx); err != nil {
		return nil, err
	}
	defer transport.Close()

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

	transport := newStreamingTransport(c.options, promptChan, true)

	if err := transport.Connect(ctx); err != nil {
		return nil, err
	}

	rawChan, err := transport.Receive(ctx)
	if err != nil {
		transport.Close()
		return nil, err
	}

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

	promptChan := make(chan map[string]any)

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

	clientTransport := newStreamingTransport(c.options, promptChan, false)

	if err := clientTransport.Connect(ctx); err != nil {
		return nil, err
	}

	sess := &session{
		transport:  clientTransport,
		logger:     c.logger.With("component", "session"),
		ctx:        ctx,
		promptChan: promptChan,
	}

	go func() {
		<-ctx.Done()
		err := sess.Close()
		if err != nil {
			c.logger.Warn("failed to close session", "error", err)
		}
	}()

	return sess, nil
}

// Close closes the client
func (c *client) Close() error {
	// Currently no persistent resources to clean up
	return nil
}

// session implements the Session interface
type session struct {
	transport  transport
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
