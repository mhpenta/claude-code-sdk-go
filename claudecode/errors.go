package claudecode

import "errors"

var (
	ErrNotInstalled     = errors.New("claude CLI not installed")
	ErrNotConnected     = errors.New("not connected")
	ErrConnectionFailed = errors.New("connection failed")
	ErrInvalidMessage   = errors.New("invalid message")
	ErrStreamClosed     = errors.New("stream closed")
)
