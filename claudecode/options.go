package claudecode

import (
	"log/slog"
	"os"
	"path/filepath"
)

// PermissionMode controls how tool execution permissions are handled
type PermissionMode string

const (
	// PermissionModeDefault prompts for dangerous tools
	PermissionModeDefault PermissionMode = "default"

	// PermissionModeAcceptEdits auto-accepts file edits
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"

	// PermissionModeBypassPermissions allows all tools (use with caution)
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// MCPServerType represents the type of MCP server
type MCPServerType string

const (
	MCPServerTypeStdio MCPServerType = "stdio"
	MCPServerTypeSSE   MCPServerType = "sse"
	MCPServerTypeHTTP  MCPServerType = "http"
)

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Type    MCPServerType     `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Options configures the Claude SDK
type Options struct {
	// SystemPrompt sets the system prompt for Claude
	SystemPrompt string

	// AppendSystemPrompt appends to the existing system prompt
	AppendSystemPrompt string

	// Model specifies which Claude model to use
	Model string

	// MaxTurns limits the number of conversation turns
	MaxTurns int

	// MaxThinkingTokens limits thinking tokens (default: 8000)
	MaxThinkingTokens int

	// PermissionMode controls tool execution permissions
	PermissionMode PermissionMode

	// AllowedTools lists tools that can be used
	AllowedTools []string

	// DisallowedTools lists tools that cannot be used
	DisallowedTools []string

	// WorkingDirectory sets the working directory for the CLI
	WorkingDirectory string

	// MCPServers configures Model Context Protocol servers
	MCPServers map[string]MCPServer

	// Continue continues a previous conversation
	Continue bool

	// Resume resumes from a specific conversation ID
	Resume string

	// Settings path to a settings file
	Settings string

	// AddDirs adds directories to the context
	AddDirs []string

	// Logger for structured logging
	Logger *slog.Logger

	// CLIPath overrides the default Claude CLI path
	CLIPath string
}

// DefaultOptions returns Options with sensible defaults
func DefaultOptions() *Options {
	return &Options{
		MaxThinkingTokens: 8000,
		PermissionMode:    PermissionModeDefault,
		Logger:            slog.Default(),
	}
}

// Option is a function that modifies Options
type Option func(*Options)

// WithLogger sets the logger
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithSystemPrompt sets the system prompt
func WithSystemPrompt(prompt string) Option {
	return func(o *Options) {
		o.SystemPrompt = prompt
	}
}

// WithModel sets the model to use
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}

// WithMaxTurns sets the maximum number of turns
func WithMaxTurns(turns int) Option {
	return func(o *Options) {
		o.MaxTurns = turns
	}
}

// WithPermissionMode sets the permission mode
func WithPermissionMode(mode PermissionMode) Option {
	return func(o *Options) {
		o.PermissionMode = mode
	}
}

// WithWorkingDirectory sets the working directory
func WithWorkingDirectory(dir string) Option {
	return func(o *Options) {
		o.WorkingDirectory = dir
	}
}

// WithAllowedTools sets the allowed tools
func WithAllowedTools(tools ...string) Option {
	return func(o *Options) {
		o.AllowedTools = tools
	}
}

// WithDisallowedTools sets the disallowed tools
func WithDisallowedTools(tools ...string) Option {
	return func(o *Options) {
		o.DisallowedTools = tools
	}
}

// WithCLIPath sets a custom CLI path
func WithCLIPath(path string) Option {
	return func(o *Options) {
		o.CLIPath = path
	}
}

// WithMCPServer adds an MCP server configuration
func WithMCPServer(name string, server MCPServer) Option {
	return func(o *Options) {
		if o.MCPServers == nil {
			o.MCPServers = make(map[string]MCPServer)
		}
		o.MCPServers[name] = server
	}
}

// WithAddDirs adds directories to the context
func WithAddDirs(dirs ...string) Option {
	return func(o *Options) {
		o.AddDirs = append(o.AddDirs, dirs...)
	}
}

// QueryOption modifies a query
type QueryOption func(*queryOptions)

type queryOptions struct {
	sessionID string
}

// WithSessionID sets the session ID for a query
func WithSessionID(id string) QueryOption {
	return func(o *queryOptions) {
		o.sessionID = id
	}
}

// SessionOption modifies a session
type SessionOption func(*sessionOptions)

type sessionOptions struct {
	initialPrompt string
}

// WithInitialPrompt sets an initial prompt for the session
func WithInitialPrompt(prompt string) SessionOption {
	return func(o *sessionOptions) {
		o.initialPrompt = prompt
	}
}

// validate checks if the options are valid
func (o *Options) validate() error {
	if o.WorkingDirectory != "" {
		if _, err := os.Stat(o.WorkingDirectory); err != nil {
			return &ClaudeError{
				Code:    "INVALID_OPTIONS",
				Message: "working directory does not exist",
				Err:     err,
			}
		}
	}

	for _, dir := range o.AddDirs {
		absPath, err := filepath.Abs(dir)
		if err != nil {
			return &ClaudeError{
				Code:    "INVALID_OPTIONS",
				Message: "invalid add directory path",
				Err:     err,
			}
		}
		if _, err := os.Stat(absPath); err != nil {
			return &ClaudeError{
				Code:    "INVALID_OPTIONS",
				Message: "add directory does not exist: " + dir,
				Err:     err,
			}
		}
	}

	return nil
}
