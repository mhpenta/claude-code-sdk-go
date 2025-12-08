package claudecode

import (
	"bufio"
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

// subprocessTransport implements transport using a subprocess
type subprocessTransport struct {
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

// newSubprocessTransport creates a new subprocess transport
func newSubprocessTransport(opts *Options) *subprocessTransport {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &subprocessTransport{
		options:     opts,
		logger:      logger.With("component", "subprocess-transport"),
		receiveDone: make(chan struct{}),
	}
}

// newStreamingTransport creates a transport for streaming mode
func newStreamingTransport(opts *Options, promptChan <-chan map[string]any, closeStdinAfterPrompt bool) *subprocessTransport {
	t := newSubprocessTransport(opts)
	t.isStreaming = true
	t.promptChan = promptChan
	t.closeStdinAfterPrompt = closeStdinAfterPrompt
	return t
}

// newOneShotTransport creates a transport for one-shot mode
func newOneShotTransport(opts *Options, prompt string) *subprocessTransport {
	t := newSubprocessTransport(opts)
	t.isStreaming = false
	t.prompt = prompt
	return t
}

// findCLI locates the Claude CLI executable
func (t *subprocessTransport) findCLI() (string, error) {
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
func (t *subprocessTransport) buildCommand() ([]string, error) {
	cliPath, err := t.findCLI()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotInstalled, err)
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

	if t.options.MaxThinkingTokens > 0 {
		args = append(args, "--max-thinking-tokens", fmt.Sprintf("%d", t.options.MaxThinkingTokens))
	}

	if len(t.options.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(t.options.DisallowedTools, ","))
	}

	if len(t.options.MCPTools) > 0 {
		args = append(args, "--mcp-tools", strings.Join(t.options.MCPTools, ","))
	}

	if t.options.Model != "" {
		args = append(args, "--model", t.options.Model)
	}

	if t.options.PermissionMode != "" {
		args = append(args, "--permission-mode", string(t.options.PermissionMode))
	}

	if t.options.PermissionPromptToolName != "" {
		args = append(args, "--permission-prompt-tool-name", t.options.PermissionPromptToolName)
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
func (t *subprocessTransport) Connect(ctx context.Context) error {
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
func (t *subprocessTransport) streamToStdin(ctx context.Context) {
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
				t.logger.Debug("error writing to stdin", slog.Any("error", err))
				return
			}
		}
	}
}

// Send sends messages to Claude
func (t *subprocessTransport) Send(ctx context.Context, messages []map[string]any) error {
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
func (t *subprocessTransport) Receive(ctx context.Context) (<-chan map[string]any, error) {
	if !t.connected.Load() {
		return nil, ErrNotConnected
	}

	msgChan := make(chan map[string]any)

	go func() {
		defer close(msgChan)
		defer close(t.receiveDone)

		scanner := bufio.NewScanner(t.stdout)
		scanner.Buffer(make([]byte, 0, maxBufferSize), maxBufferSize)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var data map[string]any
			if err := json.Unmarshal([]byte(line), &data); err != nil {
				t.logger.Debug("failed to parse JSON line",
					slog.String("line", truncate(line, 100)),
					slog.Any("error", err))
				continue
			}

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

		if err := scanner.Err(); err != nil {
			t.logger.Debug("scanner error", slog.Any("error", err))
		}

		// Wait for process to exit
		if err := t.cmd.Wait(); err != nil {
			// If we're disconnecting, exit errors are expected
			if !t.connected.Load() {
				return
			}

			// Log process failures through the structured logger
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				stderr := t.readStderr()
				t.logger.Error("subprocess exited with error",
					slog.Int("exit_code", exitErr.ExitCode()),
					slog.String("stderr", stderr))
			}
		}
	}()

	return msgChan, nil
}

// Interrupt sends an interrupt signal
func (t *subprocessTransport) Interrupt(ctx context.Context) error {
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
func (t *subprocessTransport) IsConnected() bool {
	return t.connected.Load() && t.cmd != nil && t.cmd.Process != nil
}

// Close terminates the subprocess
func (t *subprocessTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected.Load() {
		return nil
	}

	t.connected.Store(false)

	if !t.stdinClosed.Load() && t.stdin != nil {
		t.stdin.Close()
		t.stdinClosed.Store(true)
	}

	select {
	case <-t.receiveDone:
	case <-time.After(5 * time.Second):
		// Timeout waiting for receive goroutine
		if t.cmd != nil && t.cmd.Process != nil {
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
func (t *subprocessTransport) cleanup() {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.stdout != nil {
		t.stdout.Close()
	}
	if t.stderrFile != nil {
		name := t.stderrFile.Name()
		t.stderrFile.Close()
		if err := os.Remove(name); err != nil {
			t.logger.Debug("failed to remove stderr file", slog.Any("error", err))
		}
	}
}

// readStderr reads the last N lines from stderr
func (t *subprocessTransport) readStderr() string {
	if t.stderrFile == nil {
		return ""
	}

	t.stderrFile.Seek(0, 0)

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
		return fmt.Sprintf("[stderr truncated, showing last %d lines]\n%s", stderrLines, strings.Join(lines, "\n"))
	}

	return strings.Join(lines, "\n")
}

// truncate shortens a string to maxLen, appending "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
