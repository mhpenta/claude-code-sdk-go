package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mhpenta/claude-code-sdk-go/claudecode"
)

func main() {
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	fmt.Println("Comment Improvement Example")
	fmt.Println("===========================")
	fmt.Printf("Project root: %s\n\n", projectRoot)

	client, err := claudecode.New(
		claudecode.WithWorkingDirectory(projectRoot),
		claudecode.WithLogger(logger),
		claudecode.WithSystemPrompt("Improve Go code documentation and comments."),
		claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits),
		claudecode.WithAddDirs(filepath.Join(projectRoot, "claudecode")),
	)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()

	session, err := client.NewSession(context.Background())
	if err != nil {
		log.Fatal("Failed to create session:", err)
	}
	defer session.Close()

	fmt.Println("Searching for improvable comments...")

	prompt := `Search the claudecode/ directory and:
1. Find ONE comment that could be improved (clarity, context, grammar)
2. Show the current comment and file location
3. Explain why it needs improvement
4. Use the Edit tool to improve it

Focus on function/type documentation. Choose something meaningful.`

	if err := session.Send(context.Background(), prompt); err != nil {
		log.Fatal("Failed to send prompt:", err)
	}

	fmt.Println("\nAnalysis:")
	fmt.Println("---------")

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
						fmt.Printf("\n\nEdit applied to: %v\n", block.Tool.Input["file_path"])
					}
				}
			}
		case *claudecode.SystemMessage:
			if m.Subtype == "tool_use" {
				if name, ok := m.Data["name"].(string); ok && name == "Edit" {
					fmt.Println("\n[Applying edit...]")
				}
			}
		case *claudecode.ResultMessage:
			fmt.Printf("\n\nSummary:")
			fmt.Printf("\n- Duration: %dms", m.DurationMS)
			if m.TotalCostUSD != nil {
				fmt.Printf("\n- Cost: $%.4f", *m.TotalCostUSD)
			}
			fmt.Printf("\n- Success: %v", !m.IsError)
			if hasEdit {
				fmt.Println("\n\nComment successfully improved!")
			} else {
				fmt.Println("\n\nNo edits were made")
			}
			return
		}
	}
}
