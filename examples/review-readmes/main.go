package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mhpenta/claude-code-sdk-go/claudecode"
)

func main() {
	// Get the project root directory
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		log.Fatal(err)
	}

	// Create a logger that shows info level
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	fmt.Println("📚 Claude Code SDK - README Review & Update")
	fmt.Println("==========================================")
	fmt.Printf("Project root: %s\n\n", projectRoot)

	// Create client with permission to edit files
	client, err := claudecode.New(
		claudecode.WithWorkingDirectory(projectRoot),
		claudecode.WithLogger(logger),
		claudecode.WithSystemPrompt("You are a technical documentation expert for Go projects. Focus on clarity, accuracy, and completeness."),
		claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits), // Auto-accept edits
		claudecode.WithAddDirs(projectRoot), // Include entire project for context
		claudecode.WithMaxTurns(10),         // Allow multiple edits if needed
	)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()

	// Use streaming to show progress
	ctx := context.Background()
	msgChan, err := client.QueryStream(ctx, buildPrompt())
	if err != nil {
		log.Fatal("Failed to start query:", err)
	}

	fmt.Println("🔍 Reviewing README files...")
	fmt.Println("----------------------------")

	// Track edits made
	editsCount := 0
	filesEdited := make(map[string]bool)
	startTime := time.Now()

	// Process messages
	for msg := range msgChan {
		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				switch block.Type {
				case "text":
					if block.Text != nil {
						// Print Claude's analysis
						fmt.Print(*block.Text)
					}
				case "tool_use":
					if block.Tool != nil && block.Tool.Name == "Edit" {
						editsCount++
						if filepath, ok := block.Tool.Input["file_path"].(string); ok {
							filesEdited[filepath] = true
						}
					}
				}
			}

		case *claudecode.SystemMessage:
			// Show tool usage
			if m.Subtype == "tool_use" {
				if toolName, ok := m.Data["name"].(string); ok {
					switch toolName {
					case "Read":
						fmt.Println("\n📖 Reading file...")
					case "Edit":
						fmt.Println("\n✏️  Applying edit...")
					case "Grep":
						fmt.Println("\n🔎 Searching...")
					}
				}
			}

		case *claudecode.ResultMessage:
			// Final summary
			duration := time.Since(startTime)
			fmt.Println("\n\n" + strings.Repeat("=", 50))
			fmt.Println("📊 Review Summary:")
			fmt.Printf("- Duration: %.2f seconds\n", duration.Seconds())
			fmt.Printf("- Total edits made: %d\n", editsCount)
			fmt.Printf("- Files modified: %d\n", len(filesEdited))

			if len(filesEdited) > 0 {
				fmt.Println("\n📝 Modified files:")
				for file := range filesEdited {
					fmt.Printf("  - %s\n", file)
				}
			}

			if m.TotalCostUSD != nil {
				fmt.Printf("\n💰 Cost: $%.4f\n", *m.TotalCostUSD)
			}

			if !m.IsError && editsCount > 0 {
				fmt.Println("\n✅ README files successfully reviewed and updated!")
			} else if !m.IsError && editsCount == 0 {
				fmt.Println("\n✅ README files reviewed - no updates needed!")
			} else {
				fmt.Println("\n❌ Review completed with errors")
			}
		}
	}
}

func buildPrompt() string {
	return `Please review all README files in this Claude Code Go SDK project and make necessary improvements. Focus on:

1. **Main README.md** (in project root):
   - Verify all code examples work with the current API
   - Check that package names are correct (should be 'claudecode' not 'claude')
   - Ensure installation instructions are accurate
   - Update any outdated information
   - Fix any broken links or references
   - Add missing sections if needed (contributing, license, etc.)

2. **Examples README** (in examples/ directory):
   - Ensure all example descriptions match their actual functionality
   - Verify the list of examples is complete and up-to-date
   - Check that run instructions are correct
   - Add any missing examples to the list

3. **Accuracy checks**:
   - Import paths should use 'claudecode' package
   - Method names and signatures should match the implementation
   - Configuration options should be current
   - Error handling examples should use correct error types

4. **Improvements to make**:
   - Fix any typos or grammatical errors
   - Improve clarity where needed
   - Add helpful details that are missing
   - Ensure consistency in formatting and style

Please review each README file, explain what needs to be fixed, and then use the Edit tool to make the improvements. Focus on making the documentation accurate, clear, and helpful for developers using this SDK.`
}
