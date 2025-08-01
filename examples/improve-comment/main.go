package main

import (
	"context"
	"fmt"
	"github.com/mhpenta/claude-code-sdk-go/claudecode"
	"log"
	"log/slog"
	"os"
	"path/filepath"
)

func main() {
	// Get the project root directory
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		log.Fatal(err)
	}

	// Create a logger that only shows errors
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	fmt.Println("📝 Claude Code SDK - Comment Improvement Example")
	fmt.Println("==============================================")
	fmt.Printf("Project root: %s\n\n", projectRoot)

	// Create client with permission to edit files
	client, err := claudecode.New(
		claudecode.WithWorkingDirectory(projectRoot),
		claudecode.WithLogger(logger),
		claudecode.WithSystemPrompt("You are a Go documentation expert. Be concise and focus on improving code documentation."),
		claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits), // Auto-accept edits
		claudecode.WithAddDirs(filepath.Join(projectRoot, "claudecode")),
	)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()

	// Create a session for interactive feedback
	session, err := client.NewSession(context.Background())
	if err != nil {
		log.Fatal("Failed to create session:", err)
	}
	defer session.Close()

	fmt.Println("🔍 Searching for a comment to improve...")

	// Ask Claude to find and improve a comment
	prompt := `Please search through the Go SDK code in the claude/ directory and:
1. Find ONE comment that could be improved (make it more clear, add missing context, fix grammar, etc.)
2. Show me the current comment and file location
3. Explain why it needs improvement
4. Use the Edit tool to improve that comment
5. Keep the improvement concise but more helpful than the original

Focus on comments that document functions, types, or important logic. Choose something meaningful, not just a trivial comment.`

	if err := session.Send(context.Background(), prompt); err != nil {
		log.Fatal("Failed to send prompt:", err)
	}

	// Receive and display the response
	fmt.Println("\n📋 Claude's Analysis:")
	fmt.Println("--------------------")

	var hasEdit bool
	msgChan, err := session.Receive(context.Background())
	if err != nil {
		log.Fatal("Failed to receive messages:", err)
	}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				switch block.Type {
				case "text":
					if block.Text != nil {
						fmt.Print(*block.Text)
					}
				case "tool_use":
					if block.Tool != nil && block.Tool.Name == "Edit" {
						hasEdit = true
						fmt.Printf("\n\n✏️  Edit Applied to: %v\n", block.Tool.Input["file_path"])
					}
				}
			}
		case *claudecode.SystemMessage:
			if m.Subtype == "tool_use" {
				if data, ok := m.Data["name"].(string); ok && data == "Edit" {
					fmt.Println("\n🔧 Applying edit...")
				}
			}
		case *claudecode.ResultMessage:
			fmt.Printf("\n\n📊 Summary:")
			fmt.Printf("\n- Duration: %dms", m.DurationMS)
			if m.TotalCostUSD != nil {
				fmt.Printf("\n- Cost: $%.4f", *m.TotalCostUSD)
			}
			fmt.Printf("\n- Success: %v", !m.IsError)
			if hasEdit {
				fmt.Println("\n\n✅ Comment successfully improved!")
			} else {
				fmt.Println("\n\n⚠️  No edits were made")
			}
			fmt.Println()
			return
		}
	}
}
