package claudecode

import (
	"log/slog"
	"os"
	"testing"
)

func TestOptionsComplete(t *testing.T) {
	opts := DefaultOptions()
	options := []Option{
		WithLogger(slog.New(slog.NewTextHandler(os.Stderr, nil))),
		WithSystemPrompt("test system prompt"),
		WithAppendSystemPrompt("append this"),
		WithModel("claude-3-haiku"),
		WithMaxTurns(5),
		WithMaxThinkingTokens(4000),
		WithPermissionMode(PermissionModeAcceptEdits),
		WithPermissionPromptToolName("test-tool"),
		WithAllowedTools("Read", "Write"),
		WithDisallowedTools("Bash"),
		WithMCPTools("filesystem", "database"),
		WithWorkingDirectory("."),
		WithMCPServer("test", MCPServer{
			Type:    MCPServerTypeStdio,
			Command: "test-command",
		}),
		WithContinue(true),
		WithResume("test-conversation-id"),
		WithSettings("/path/to/settings.json"),
		WithAddDirs("./src", "./docs"),
		WithCLIPath("/custom/claude"),
	}

	for _, opt := range options {
		opt(opts)
	}

	if opts.SystemPrompt != "test system prompt" {
		t.Errorf("SystemPrompt not set correctly")
	}
	if opts.AppendSystemPrompt != "append this" {
		t.Errorf("AppendSystemPrompt not set correctly")
	}
	if opts.MaxThinkingTokens != 4000 {
		t.Errorf("MaxThinkingTokens not set correctly")
	}
	if opts.PermissionPromptToolName != "test-tool" {
		t.Errorf("PermissionPromptToolName not set correctly")
	}
	if len(opts.MCPTools) != 2 {
		t.Errorf("MCPTools not set correctly")
	}
	if !opts.Continue {
		t.Errorf("Continue not set correctly")
	}
	if opts.Resume != "test-conversation-id" {
		t.Errorf("Resume not set correctly")
	}
}
