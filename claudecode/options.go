package claudecode

import (
	"encoding/json"
	"fmt"
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

	// PermissionModePlan enables plan-only mode
	PermissionModePlan PermissionMode = "plan"

	// PermissionModeBypassPermissions allows all operations (use with caution)
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// SdkBeta represents SDK beta features
type SdkBeta string

const (
	// SdkBetaContext1M enables extended context window
	SdkBetaContext1M SdkBeta = "context-1m-2025-08-07"
)

// SettingSource represents where settings should be loaded from
type SettingSource string

const (
	SettingSourceUser    SettingSource = "user"
	SettingSourceProject SettingSource = "project"
	SettingSourceLocal   SettingSource = "local"
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

// AgentModel represents the model options for an agent
type AgentModel string

const (
	AgentModelSonnet  AgentModel = "sonnet"
	AgentModelOpus    AgentModel = "opus"
	AgentModelHaiku   AgentModel = "haiku"
	AgentModelInherit AgentModel = "inherit"
)

// AgentDefinition represents a custom agent configuration
type AgentDefinition struct {
	Description string     `json:"description"`
	Prompt      string     `json:"prompt"`
	Tools       []string   `json:"tools,omitempty"`
	Model       AgentModel `json:"model,omitempty"`
}

// SandboxNetworkConfig represents network configuration for sandbox
type SandboxNetworkConfig struct {
	AllowUnixSockets    []string `json:"allowUnixSockets,omitempty"`
	AllowAllUnixSockets bool     `json:"allowAllUnixSockets,omitempty"`
	AllowLocalBinding   bool     `json:"allowLocalBinding,omitempty"`
	HTTPProxyPort       int      `json:"httpProxyPort,omitempty"`
	SOCKSProxyPort      int      `json:"socksProxyPort,omitempty"`
}

// SandboxIgnoreViolations represents violations to ignore in sandbox
type SandboxIgnoreViolations struct {
	File    []string `json:"file,omitempty"`
	Network []string `json:"network,omitempty"`
}

// SandboxSettings represents sandbox configuration for bash command isolation
type SandboxSettings struct {
	Enabled                    bool                     `json:"enabled,omitempty"`
	AutoAllowBashIfSandboxed   bool                     `json:"autoAllowBashIfSandboxed,omitempty"`
	ExcludedCommands           []string                 `json:"excludedCommands,omitempty"`
	AllowUnsandboxedCommands   bool                     `json:"allowUnsandboxedCommands,omitempty"`
	Network                    *SandboxNetworkConfig    `json:"network,omitempty"`
	IgnoreViolations           *SandboxIgnoreViolations `json:"ignoreViolations,omitempty"`
	EnableWeakerNestedSandbox  bool                     `json:"enableWeakerNestedSandbox,omitempty"`
}

// PluginType represents the type of plugin
type PluginType string

const (
	PluginTypeLocal PluginType = "local"
)

// PluginConfig represents a plugin configuration
type PluginConfig struct {
	Type PluginType `json:"type"`
	Path string     `json:"path"`
}

// SystemPromptPreset represents a system prompt preset configuration
type SystemPromptPreset struct {
	Type   string `json:"type"`   // "preset"
	Preset string `json:"preset"` // "claude_code"
	Append string `json:"append,omitempty"`
}

// ToolsPreset represents a tools preset configuration
type ToolsPreset struct {
	Type   string `json:"type"`   // "preset"
	Preset string `json:"preset"` // "claude_code"
}

// OutputFormat represents structured output configuration
type OutputFormat struct {
	Type   string         `json:"type"`             // "json_schema"
	Schema map[string]any `json:"schema,omitempty"` // JSON schema definition
}

// AssistantMessageError represents possible error types from the assistant
type AssistantMessageError string

const (
	AssistantMessageErrorAuthFailed     AssistantMessageError = "authentication_failed"
	AssistantMessageErrorBillingError   AssistantMessageError = "billing_error"
	AssistantMessageErrorRateLimit      AssistantMessageError = "rate_limit"
	AssistantMessageErrorInvalidRequest AssistantMessageError = "invalid_request"
	AssistantMessageErrorServerError    AssistantMessageError = "server_error"
	AssistantMessageErrorUnknown        AssistantMessageError = "unknown"
)

// Options configures the Claude SDK
type Options struct {
	// SystemPrompt sets the system prompt for Claude
	// Can be a string or use WithSystemPromptPreset for preset configuration
	SystemPrompt string

	// SystemPromptPreset configures a preset system prompt
	SystemPromptPreset *SystemPromptPreset

	// AppendSystemPrompt appends to the existing system prompt (deprecated, use SystemPromptPreset.Append)
	AppendSystemPrompt string

	// Model specifies which Claude model to use
	Model string

	// FallbackModel specifies a fallback model if the primary is unavailable
	FallbackModel string

	// MaxTurns limits the number of conversation turns
	MaxTurns int

	// MaxThinkingTokens limits thinking tokens (default: 8000)
	MaxThinkingTokens int

	// MaxBudgetUSD sets a spending limit in USD
	MaxBudgetUSD *float64

	// PermissionMode controls tool execution permissions
	PermissionMode PermissionMode

	// PermissionPromptToolName specifies tool name for permission prompts
	PermissionPromptToolName string

	// Tools specifies available tools (list of names, empty list to disable built-ins, or preset)
	Tools []string

	// ToolsPreset configures a tools preset
	ToolsPreset *ToolsPreset

	// AllowedTools lists tools that can be used
	AllowedTools []string

	// DisallowedTools lists tools that cannot be used
	DisallowedTools []string

	// MCPTools lists MCP tools that can be used
	MCPTools []string

	// WorkingDirectory sets the working directory for the CLI
	WorkingDirectory string

	// MCPServers configures Model Context Protocol servers
	MCPServers map[string]MCPServer

	// Continue continues a previous conversation
	Continue bool

	// Resume resumes from a specific conversation ID
	Resume string

	// ForkSession creates a new session when resuming instead of continuing
	ForkSession bool

	// Settings path to a settings file
	Settings string

	// SettingSources controls which settings to load (user, project, local)
	SettingSources []SettingSource

	// AddDirs adds directories to the context
	AddDirs []string

	// Env sets additional environment variables
	Env map[string]string

	// ExtraArgs passes arbitrary CLI flags
	ExtraArgs map[string]*string

	// Betas enables SDK beta features
	Betas []SdkBeta

	// Agents defines custom agents
	Agents map[string]AgentDefinition

	// Sandbox configures bash command sandboxing
	Sandbox *SandboxSettings

	// Plugins configures custom plugins
	Plugins []PluginConfig

	// OutputFormat configures structured output
	OutputFormat *OutputFormat

	// IncludePartialMessages enables streaming of partial message updates
	IncludePartialMessages bool

	// EnableFileCheckpointing enables file change tracking for rewind support
	EnableFileCheckpointing bool

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

// WithSystemPromptPreset sets a preset system prompt configuration
func WithSystemPromptPreset(preset SystemPromptPreset) Option {
	return func(o *Options) {
		o.SystemPromptPreset = &preset
	}
}

// WithModel sets the model to use
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}

// WithFallbackModel sets a fallback model
func WithFallbackModel(model string) Option {
	return func(o *Options) {
		o.FallbackModel = model
	}
}

// WithMaxTurns sets the maximum number of turns
func WithMaxTurns(turns int) Option {
	return func(o *Options) {
		o.MaxTurns = turns
	}
}

// WithMaxThinkingTokens sets the maximum thinking tokens
func WithMaxThinkingTokens(tokens int) Option {
	return func(o *Options) {
		o.MaxThinkingTokens = tokens
	}
}

// WithMaxBudgetUSD sets a spending limit in USD
func WithMaxBudgetUSD(budget float64) Option {
	return func(o *Options) {
		o.MaxBudgetUSD = &budget
	}
}

// WithPermissionMode sets the permission mode
func WithPermissionMode(mode PermissionMode) Option {
	return func(o *Options) {
		o.PermissionMode = mode
	}
}

// WithPermissionPromptToolName sets the tool name for permission prompts
func WithPermissionPromptToolName(toolName string) Option {
	return func(o *Options) {
		o.PermissionPromptToolName = toolName
	}
}

// WithTools sets the available tools (pass empty slice to disable all built-in tools)
func WithTools(tools ...string) Option {
	return func(o *Options) {
		o.Tools = tools
	}
}

// WithToolsPreset sets a tools preset configuration
func WithToolsPreset(preset ToolsPreset) Option {
	return func(o *Options) {
		o.ToolsPreset = &preset
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

// WithMCPTools sets the MCP tools that can be used
func WithMCPTools(tools ...string) Option {
	return func(o *Options) {
		o.MCPTools = tools
	}
}

// WithWorkingDirectory sets the working directory
func WithWorkingDirectory(dir string) Option {
	return func(o *Options) {
		o.WorkingDirectory = dir
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

// WithAppendSystemPrompt appends to the system prompt (deprecated, use WithSystemPromptPreset)
func WithAppendSystemPrompt(prompt string) Option {
	return func(o *Options) {
		o.AppendSystemPrompt = prompt
	}
}

// WithContinue enables continuing a previous conversation
func WithContinue() Option {
	return func(o *Options) {
		o.Continue = true
	}
}

// WithResume resumes from a specific conversation ID
func WithResume(conversationID string) Option {
	return func(o *Options) {
		o.Resume = conversationID
	}
}

// WithForkSession enables forking when resuming a session
func WithForkSession() Option {
	return func(o *Options) {
		o.ForkSession = true
	}
}

// WithSettings sets the path to a settings file
func WithSettings(path string) Option {
	return func(o *Options) {
		o.Settings = path
	}
}

// WithSettingSources sets which settings to load
func WithSettingSources(sources ...SettingSource) Option {
	return func(o *Options) {
		o.SettingSources = sources
	}
}

// WithEnv sets additional environment variables
func WithEnv(env map[string]string) Option {
	return func(o *Options) {
		if o.Env == nil {
			o.Env = make(map[string]string)
		}
		for k, v := range env {
			o.Env[k] = v
		}
	}
}

// WithExtraArg adds an extra CLI argument
// Use nil for value to pass flag without value (e.g., --flag instead of --flag=value)
func WithExtraArg(name string, value *string) Option {
	return func(o *Options) {
		if o.ExtraArgs == nil {
			o.ExtraArgs = make(map[string]*string)
		}
		o.ExtraArgs[name] = value
	}
}

// WithBetas enables SDK beta features
func WithBetas(betas ...SdkBeta) Option {
	return func(o *Options) {
		o.Betas = append(o.Betas, betas...)
	}
}

// WithAgent adds a custom agent definition
func WithAgent(name string, agent AgentDefinition) Option {
	return func(o *Options) {
		if o.Agents == nil {
			o.Agents = make(map[string]AgentDefinition)
		}
		o.Agents[name] = agent
	}
}

// WithSandbox sets sandbox configuration
func WithSandbox(sandbox SandboxSettings) Option {
	return func(o *Options) {
		o.Sandbox = &sandbox
	}
}

// WithPlugin adds a plugin configuration
func WithPlugin(plugin PluginConfig) Option {
	return func(o *Options) {
		o.Plugins = append(o.Plugins, plugin)
	}
}

// WithOutputFormat sets structured output configuration
func WithOutputFormat(format OutputFormat) Option {
	return func(o *Options) {
		o.OutputFormat = &format
	}
}

// WithIncludePartialMessages enables streaming of partial message updates
func WithIncludePartialMessages() Option {
	return func(o *Options) {
		o.IncludePartialMessages = true
	}
}

// WithEnableFileCheckpointing enables file change tracking
func WithEnableFileCheckpointing() Option {
	return func(o *Options) {
		o.EnableFileCheckpointing = true
	}
}

// WithCLIPath sets a custom CLI path
func WithCLIPath(path string) Option {
	return func(o *Options) {
		o.CLIPath = path
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
			return fmt.Errorf("working directory does not exist: %w", err)
		}
	}

	for _, dir := range o.AddDirs {
		absPath, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("invalid add directory path %q: %w", dir, err)
		}
		if _, err := os.Stat(absPath); err != nil {
			return fmt.Errorf("add directory does not exist %q: %w", dir, err)
		}
	}

	return nil
}

// MarshalSystemPrompt marshals the system prompt configuration to JSON
func (o *Options) MarshalSystemPrompt() ([]byte, error) {
	if o.SystemPromptPreset != nil {
		return json.Marshal(o.SystemPromptPreset)
	}
	return nil, nil
}

// MarshalTools marshals the tools configuration to JSON
func (o *Options) MarshalTools() ([]byte, error) {
	if o.ToolsPreset != nil {
		return json.Marshal(o.ToolsPreset)
	}
	if o.Tools != nil {
		return json.Marshal(o.Tools)
	}
	return nil, nil
}
