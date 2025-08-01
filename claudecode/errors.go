package claudecode

import (
	"errors"
	"fmt"
)

// Sentinel errors for the Claude SDK
var (
	// ErrClaudeNotInstalled is returned when the Claude CLI is not found
	ErrClaudeNotInstalled = errors.New("claude-code: CLI not installed")

	// ErrNotConnected is returned when trying to use a disconnected client
	ErrNotConnected = errors.New("claude-code: not connected")

	// ErrConnectionFailed is returned when connection to Claude fails
	ErrConnectionFailed = errors.New("claude-code: connection failed")

	// ErrInvalidMessage is returned when a message is invalid
	ErrInvalidMessage = errors.New("claude-code: invalid message")

	// ErrJSONDecode is returned when JSON decoding fails
	ErrJSONDecode = errors.New("claude-code: JSON decode error")

	// ErrProcessExited is returned when the Claude process exits unexpectedly
	ErrProcessExited = errors.New("claude-code: process exited unexpectedly")

	// ErrInterrupted is returned when an operation is interrupted
	ErrInterrupted = errors.New("claude-code: operation interrupted")

	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("claude-code: operation timed out")

	// ErrStreamClosed is returned when trying to use a closed stream
	ErrStreamClosed = errors.New("claude-code: stream closed")
)

// ClaudeError provides structured error information
type ClaudeError struct {
	Code    string
	Message string
	Err     error
}

// Error implements the error interface
func (e *ClaudeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("claude-code %s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("claude-code %s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *ClaudeError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is support
func (e *ClaudeError) Is(target error) bool {
	return e.Err != nil && errors.Is(e.Err, target)
}

// ProcessError contains information about process failures
type ProcessError struct {
	ExitCode int
	Stderr   string
	Err      error
}

// Error implements the error interface
func (e *ProcessError) Error() string {
	msg := fmt.Sprintf("claude-code: process failed with exit code %d", e.ExitCode)
	if e.Stderr != "" {
		msg += fmt.Sprintf("\nError output: %s", e.Stderr)
	}
	if e.Err != nil {
		msg += fmt.Sprintf("\nUnderlying error: %v", e.Err)
	}
	return msg
}

// Unwrap returns the underlying error
func (e *ProcessError) Unwrap() error {
	return e.Err
}

// JSONDecodeError contains information about JSON parsing failures
type JSONDecodeError struct {
	Data []byte
	Err  error
}

// Error implements the error interface
func (e *JSONDecodeError) Error() string {
	preview := string(e.Data)
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	return fmt.Sprintf("claude-code: failed to decode JSON: %v\nData: %s", e.Err, preview)
}

// Unwrap returns the underlying error
func (e *JSONDecodeError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is support
func (e *JSONDecodeError) Is(target error) bool {
	return errors.Is(target, ErrJSONDecode)
}
