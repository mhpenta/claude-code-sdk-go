package claudecode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	maxBufferSize = 1024 * 1024 // 1MB buffer limit
	stderrLines   = 100         // Keep last N stderr lines
)

// SubprocessTransport implements Transport using subprocess
type SubprocessTransport struct {
	options    *Options
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderrFile *os.File
	connected  atomic.Bool
	logger     *slog.Logger

	// Streaming support
	isStreaming           bool
	prompt                string
	promptChan            <-chan map[string]any
	closeStdinAfterPrompt bool

	// Synchronization
	mu          sync.Mutex
	receiveDone chan struct{}
	stdinClosed atomic.Bool
}

// NewSubprocessTransport creates a new subprocess transport
func NewSubprocessTransport(opts *Options) *SubprocessTransport {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &SubprocessTransport{
		options:     opts,
		logger:      logger.With("component", "subprocess-transport"),
		receiveDone: make(chan struct{}),
	}
}

// NewStreamingTransport creates a transport for streaming mode
func NewStreamingTransport(opts *Options, promptChan <-chan map[string]any, closeStdinAfterPrompt bool) *SubprocessTransport {
	t := NewSubprocessTransport(opts)
	t.isStreaming = true
	t.promptChan = promptChan
	t.closeStdinAfterPrompt = closeStdinAfterPrompt
	return t
}

// NewOneShotTransport creates a transport for one-shot mode
func NewOneShotTransport(opts *Options, prompt string) *SubprocessTransport {
	t := NewSubprocessTransport(opts)
	t.isStreaming = false
	t.prompt = prompt
	return t
}

// findCLI locates the Claude CLI executable using the following priority:
// 1. Custom path from options.CLIPath (if provided)
// 2. System PATH via exec.LookPath
// 3. Common installation locations (npm global, local bins, node_modules)
// Returns the full path to the executable, or a detailed error with installation
// instructions if not found. The error messages include specific guidance for
// installing Node.js (if missing) and the Claude Code package.
func (t *SubprocessTransport) findCLI() (string, error) {
	// Check if custom path is provided
	if t.options.CLIPath != "" {
		if _, err := os.Stat(t.options.CLIPath); err == nil {
			return t.options.CLIPath, nil
		}
		return "", fmt.Errorf("claude CLI not found at specified path: %s", t.options.CLIPath)
	}

	if path, err := exec.LookPath("claude"); err == nil {
		return path, nil
	}

	// Check common locations
	locations := []string{
		filepath.Join(os.Getenv("HOME"), ".npm-global/bin/claude"),
		"/usr/local/bin/claude",
		filepath.Join(os.Getenv("HOME"), ".local/bin/claude"),
		filepath.Join(os.Getenv("HOME"), "node_modules/.bin/claude"),
		filepath.Join(os.Getenv("HOME"), ".yarn/bin/claude"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	// Check if Node.js is installed
	if _, err := exec.LookPath("node"); err != nil {
		return "", errors.New("Claude Code requires Node.js, which is not installed.\n\n" +
			"Install Node.js from: https://nodejs.org/\n" +
			"\nAfter installing Node.js, install Claude Code:\n" +
			"  npm install -g @anthropic-ai/claude-code")
	}

	return "", errors.New("Claude Code not found. Install with:\n" +
		"  npm install -g @anthropic-ai/claude-code\n" +
		"\nIf already installed locally, try:\n" +
		"  export PATH=\"$HOME/node_modules/.bin:$PATH\"\n" +
		"\nOr specify the path when creating the client:\n" +
		"  New(WithCLIPath(\"/path/to/claude\"))")
}

// buildCommand constructs the CLI command with arguments
func (t *SubprocessTransport) buildCommand() ([]string, error) {
	cliPath, err := t.findCLI()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrClaudeNotInstalled, err)
	}

	args := []string{cliPath, "--output-format", "stream-json", "--verbose"}

	if t.options.SystemPrompt != "" {
		args = append(args, "--system-prompt", t.options.SystemPrompt)
	}

	if t.options.AppendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", t.options.AppendSystemPrompt)
	}

	if len(t.options.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(t.options.AllowedTools, ","))
	}

	if t.options.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", t.options.MaxTurns))
	}

	if len(t.options.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(t.options.DisallowedTools, ","))
	}

	if t.options.Model != "" {
		args = append(args, "--model", t.options.Model)
	}

	if t.options.PermissionMode != "" {
		args = append(args, "--permission-mode", string(t.options.PermissionMode))
	}

	if t.options.Continue {
		args = append(args, "--continue")
	}

	if t.options.Resume != "" {
		args = append(args, "--resume", t.options.Resume)
	}

	if t.options.Settings != "" {
		args = append(args, "--settings", t.options.Settings)
	}

	for _, dir := range t.options.AddDirs {
		args = append(args, "--add-dir", dir)
	}

	if len(t.options.MCPServers) > 0 {
		mcpConfig := map[string]any{"mcpServers": t.options.MCPServers}
		configJSON, err := json.Marshal(mcpConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal MCP config: %w", err)
		}
		args = append(args, "--mcp-config", string(configJSON))
	}

	// Add prompt handling based on mode
	if t.isStreaming {
		args = append(args, "--input-format", "stream-json")
	} else {
		args = append(args, "--print", t.prompt)
	}

	return args, nil
}

// Connect establishes the subprocess connection
func (t *SubprocessTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected.Load() {
		return nil
	}

	cmdArgs, err := t.buildCommand()
	if err != nil {
		return err
	}

	// Create temp file for stderr
	t.stderrFile, err = os.CreateTemp("", "claude_stderr_*.log")
	if err != nil {
		return fmt.Errorf("%w: failed to create stderr file: %v", ErrConnectionFailed, err)
	}

	// Build command
	t.cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	t.cmd.Env = append(os.Environ(), "CLAUDE_CODE_ENTRYPOINT=sdk-go")

	if t.options.WorkingDirectory != "" {
		t.cmd.Dir = t.options.WorkingDirectory
	}

	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		t.cleanup()
		return fmt.Errorf("%w: failed to create stdin pipe: %v", ErrConnectionFailed, err)
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		t.cleanup()
		return fmt.Errorf("%w: failed to create stdout pipe: %v", ErrConnectionFailed, err)
	}

	t.cmd.Stderr = t.stderrFile

	if err := t.cmd.Start(); err != nil {
		t.cleanup()
		if t.options.WorkingDirectory != "" {
			if _, statErr := os.Stat(t.options.WorkingDirectory); statErr != nil {
				return fmt.Errorf("%w: working directory does not exist: %s", ErrConnectionFailed, t.options.WorkingDirectory)
			}
		}
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	t.connected.Store(true)
	t.logger.Debug("subprocess started", slog.Int("pid", t.cmd.Process.Pid))

	if t.isStreaming && t.promptChan != nil {
		go t.streamToStdin(ctx)
	} else if !t.isStreaming {
		// Close stdin immediately for one-shot mode
		t.stdin.Close()
		t.stdinClosed.Store(true)
	}

	return nil
}

// streamToStdin handles streaming prompts to stdin
func (t *SubprocessTransport) streamToStdin(ctx context.Context) {
	defer func() {
		if !t.stdinClosed.Load() {
			t.stdin.Close()
			t.stdinClosed.Store(true)
		}
	}()

	encoder := json.NewEncoder(t.stdin)

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-t.promptChan:
			if !ok {
				if t.closeStdinAfterPrompt {
					return
				}
				// Channel closed but keep stdin open for interactive mode
				select {
				case <-ctx.Done():
					return
				case <-t.receiveDone:
					return
				}
			}

			if err := encoder.Encode(msg); err != nil {
				if t.logger != nil {
					t.logger.Debug("error writing to stdin", slog.Any("error", err))
				}
				return
			}
		}
	}
}

// Send sends messages to Claude
func (t *SubprocessTransport) Send(ctx context.Context, messages []map[string]any) error {
	if !t.isStreaming {
		return errors.New("send only works in streaming mode")
	}

	if !t.connected.Load() {
		return ErrNotConnected
	}

	if t.stdinClosed.Load() {
		return errors.New("stdin closed - stream may have ended")
	}

	encoder := json.NewEncoder(t.stdin)
	for _, msg := range messages {
		if err := encoder.Encode(msg); err != nil {
			return fmt.Errorf("failed to encode message: %w", err)
		}
	}

	return nil
}

// Receive returns a channel for receiving messages
func (t *SubprocessTransport) Receive(ctx context.Context) (<-chan map[string]any, error) {
	if !t.connected.Load() {
		return nil, ErrNotConnected
	}

	msgChan := make(chan map[string]any)

	go func() {
		defer close(msgChan)
		defer close(t.receiveDone)

		scanner := bufio.NewScanner(t.stdout)
		scanner.Buffer(make([]byte, 0, maxBufferSize), maxBufferSize)

		jsonBuffer := &bytes.Buffer{}

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			// Handle multiple JSON objects on one line
			lines := strings.Split(line, "\n")
			for _, jsonLine := range lines {
				jsonLine = strings.TrimSpace(jsonLine)
				if jsonLine == "" {
					continue
				}

				jsonBuffer.WriteString(jsonLine)

				// Check buffer size
				if jsonBuffer.Len() > maxBufferSize {
					if t.logger != nil {
						t.logger.Error("JSON buffer exceeded maximum size",
							slog.Int("size", jsonBuffer.Len()))
					}
					jsonBuffer.Reset()
					continue
				}

				// Try to parse JSON
				var data map[string]any
				if err := json.Unmarshal(jsonBuffer.Bytes(), &data); err == nil {
					jsonBuffer.Reset()

					// Skip control responses
					if data["type"] == "control_response" {
						continue
					}

					select {
					case msgChan <- data:
					case <-ctx.Done():
						return
					}
				}
				// If parse fails, continue accumulating
			}
		}

		if err := scanner.Err(); err != nil {
			if t.logger != nil {
				t.logger.Debug("scanner error", slog.Any("error", err))
			}
		}

		defer func() {
			if r := recover(); r != nil {
				// If we panic here, just silently ignore it
				// The process is exiting anyway
				fmt.Fprintf(os.Stderr, "recovered from panic during subprocess exit: %v\n", r)
			}
		}()

		// Wait for process to exit
		err := t.cmd.Wait()
		if err != nil {
			// Only log actual errors, not normal exits
			// Check if this is a real error or just normal termination
			if !t.connected.Load() {
				// We're disconnecting, this is expected
				return
			}

			// Check if it's an exit error with a non-zero code
			if exitErr, ok := err.(*exec.ExitError); ok {
				if t.connected.Load() {
					stderr := t.readStderr()
					if stderr != "" {
						fmt.Fprintf(os.Stderr, "Claude Code failed with exit status %d\n", exitErr.ExitCode())
						fmt.Fprintf(os.Stderr, "Error details:\n%s\n", stderr)
					} else {
						fmt.Fprintf(os.Stderr, "subprocess exited with error: %v\n", err)
					}
				}
			} else {
				// This might be important, so log it to stderr
				fmt.Fprintf(os.Stderr, "subprocess wait error: %v\n", err)
			}
		}
	}()

	return msgChan, nil
}

// Interrupt sends an interrupt signal
func (t *SubprocessTransport) Interrupt(ctx context.Context) error {
	if !t.isStreaming {
		return errors.New("interrupt requires streaming mode")
	}

	if !t.connected.Load() || t.stdinClosed.Load() {
		return ErrNotConnected
	}

	controlReq := map[string]any{
		"type":       "control_request",
		"request_id": fmt.Sprintf("req_%d", time.Now().UnixNano()),
		"request": map[string]string{
			"subtype": "interrupt",
		},
	}

	encoder := json.NewEncoder(t.stdin)
	return encoder.Encode(controlReq)
}

// IsConnected returns true if connected
func (t *SubprocessTransport) IsConnected() bool {
	return t.connected.Load() && (t.cmd != nil && t.cmd.Process != nil)
}

// Close terminates the subprocess
func (t *SubprocessTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected.Load() {
		return nil
	}

	t.connected.Store(false)

	// Close stdin if not already closed
	if !t.stdinClosed.Load() && t.stdin != nil {
		t.stdin.Close()
		t.stdinClosed.Store(true)
	}

	// Wait for receive goroutine to finish first
	// This ensures we don't have double Wait() calls
	select {
	case <-t.receiveDone:
		// Receive goroutine has finished
	case <-time.After(5 * time.Second):
		// Timeout waiting for receive goroutine
		if t.cmd != nil && t.cmd.Process != nil {
			// Force terminate
			err := t.cmd.Process.Kill()
			if err != nil {
				return err
			}
		}
	}

	t.cleanup()
	return nil
}

// cleanup removes temporary files and closes handles
func (t *SubprocessTransport) cleanup() {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.stdout != nil {
		t.stdout.Close()
	}
	if t.stderrFile != nil {
		name := t.stderrFile.Name()
		t.stderrFile.Close()
		os.Remove(name)
	}
}

// readStderr reads the last N lines from stderr
func (t *SubprocessTransport) readStderr() string {
	if t.stderrFile == nil {
		return ""
	}

	// Seek to beginning
	t.stderrFile.Seek(0, 0)

	// Read all lines into a circular buffer
	lines := make([]string, 0, stderrLines)
	scanner := bufio.NewScanner(t.stderrFile)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
			if len(lines) > stderrLines {
				lines = lines[1:]
			}
		}
	}

	if len(lines) == stderrLines {
		return fmt.Sprintf("[stderr truncated, showing last %d lines]\n%s",
			stderrLines, strings.Join(lines, "\n"))
	}

	return strings.Join(lines, "\n")
}
