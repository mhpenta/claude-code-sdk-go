package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

	fmt.Println("README Review & Update")
	fmt.Println("======================")
	fmt.Printf("Project root: %s\n\n", projectRoot)

	client, err := claudecode.New(
		claudecode.WithWorkingDirectory(projectRoot),
		claudecode.WithLogger(logger),
		claudecode.WithSystemPrompt("Review and improve technical documentation for clarity and accuracy."),
		claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits),
		claudecode.WithAddDirs(projectRoot),
		claudecode.WithMaxTurns(10),
	)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()

	ctx := context.Background()
	msgChan, err := client.QueryStream(ctx, buildPrompt())
	if err != nil {
		log.Fatal("Failed to start query:", err)
	}

	fmt.Println("Reviewing README files...")
	fmt.Println(strings.Repeat("-", 40))

	editsCount := 0
	filesEdited := make(map[string]bool)

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
						editsCount++
						if path, ok := block.Tool.Input["file_path"].(string); ok {
							filesEdited[path] = true
						}
					}
				}
			}

		case *claudecode.SystemMessage:
			if m.Subtype == "tool_use" {
				if toolName, ok := m.Data["name"].(string); ok {
					switch toolName {
					case "Read":
						fmt.Println("\n[Reading file...]")
					case "Edit":
						fmt.Println("\n[Applying edit...]")
					case "Grep":
						fmt.Println("\n[Searching...]")
					}
				}
			}

		case *claudecode.ResultMessage:
			fmt.Println("\n\n" + strings.Repeat("=", 50))
			fmt.Println("Summary:")
			fmt.Printf("- Duration: %dms\n", m.DurationMS)
			fmt.Printf("- Edits: %d\n", editsCount)
			fmt.Printf("- Files modified: %d\n", len(filesEdited))

			if len(filesEdited) > 0 {
				fmt.Println("\nModified:")
				for file := range filesEdited {
					fmt.Printf("  - %s\n", file)
				}
			}

			if m.TotalCostUSD != nil {
				fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
			}

			if m.IsError {
				fmt.Println("\nCompleted with errors")
			} else if editsCount > 0 {
				fmt.Println("\nREADME files updated!")
			} else {
				fmt.Println("\nNo updates needed")
			}
		}
	}
}

func buildPrompt() string {
	return `Review README files in this Go SDK project and fix any issues:

1. Main README.md:
   - Verify code examples match the current API
   - Check package names (should be 'claudecode')
   - Update outdated information

2. Examples README (if exists):
   - Ensure descriptions match functionality
   - Verify run instructions

3. Fix:
   - Typos and grammar
   - Incorrect method signatures
   - Wrong error type names

Use the Edit tool to make corrections.`
}
