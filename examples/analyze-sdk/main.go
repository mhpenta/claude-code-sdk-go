package main

import (
	"context"
	"fmt"
	"github.com/mhpenta/claude-code-sdk-go/claudecode"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Get the project root directory (two levels up from examples/self-analysis)
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		log.Fatal(err)
	}

	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Only show warnings and errors
	}))

	fmt.Println("🔍 Claude Code SDK Self-Analysis")
	fmt.Println("================================")
	fmt.Printf("Project root: %s\n\n", projectRoot)

	// Create client with project directory context
	client, err := claudecode.New(
		claudecode.WithWorkingDirectory(projectRoot),
		claudecode.WithLogger(logger),
		claudecode.WithSystemPrompt("You are a Go code reviewer. Be concise and focus on important observations."),
		claudecode.WithPermissionMode(claudecode.PermissionModeDefault),
		claudecode.WithAddDirs(filepath.Join(projectRoot, "claudecode")),
	)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()

	// Analyze the SDK architecture
	fmt.Println("📊 Analyzing SDK Architecture...")
	if err := analyzeArchitecture(client); err != nil {
		log.Fatal("Architecture analysis failed:", err)
	}

	// Review code quality
	fmt.Println("\n💎 Reviewing Code Quality...")
	if err := reviewCodeQuality(client); err != nil {
		log.Fatal("Code quality review failed:", err)
	}

	// Suggest improvements
	fmt.Println("\n🚀 Suggesting Improvements...")
	if err := suggestImprovements(client); err != nil {
		log.Fatal("Improvement suggestions failed:", err)
	}
}

func analyzeArchitecture(client claudecode.Client) error {
	ctx := context.Background()

	prompt := `Please analyze the architecture of this Claude Code Go SDK in the claude/ directory. 
Focus on:
1. Overall design patterns used
2. Interface design and abstraction
3. How it compares to the Python implementation
4. Strengths of the current architecture

Be concise - limit your response to key observations.`

	messages, err := client.Query(ctx, prompt)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	printResponse(messages)
	return nil
}

func reviewCodeQuality(client claudecode.Client) error {
	ctx := context.Background()

	prompt := `Review the code quality of the Go SDK implementation in the claude/ directory.
Look for:
1. Go idioms and best practices
2. Error handling patterns
3. Concurrency safety
4. Code organization

Highlight both good practices and any concerns.`

	messages, err := client.Query(ctx, prompt)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	printResponse(messages)
	return nil
}

func suggestImprovements(client claudecode.Client) error {
	ctx := context.Background()

	prompt := `Based on your analysis of the Claude Code Go SDK, suggest the top 3-5 improvements that would:
1. Make the SDK more robust
2. Improve the developer experience
3. Better align with Go best practices

For each suggestion, briefly explain why it's important.`

	// Use streaming for this one to show real-time output
	msgChan, err := client.QueryStream(ctx, prompt)
	if err != nil {
		return fmt.Errorf("query stream failed: %w", err)
	}

	printStreamingResponse(msgChan)
	return nil
}

func printResponse(messages []claudecode.Message) {
	for _, msg := range messages {
		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				if block.Type == "text" && block.Text != nil {
					fmt.Println(*block.Text)
				}
			}
		case *claudecode.ResultMessage:
			fmt.Printf("\n⏱️  Duration: %dms", m.DurationMS)
			if m.TotalCostUSD != nil {
				fmt.Printf(" | 💰 Cost: $%.4f", *m.TotalCostUSD)
			}
			fmt.Println()
		}
	}
}

func printStreamingResponse(msgChan <-chan claudecode.Message) {
	var assistantOutput strings.Builder

	for msg := range msgChan {
		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				if block.Type == "text" && block.Text != nil {
					assistantOutput.WriteString(*block.Text)
					fmt.Print(*block.Text)
				}
			}
		case *claudecode.ResultMessage:
			// Print summary at the end
			fmt.Printf("\n\n⏱️  Duration: %dms", m.DurationMS)
			if m.TotalCostUSD != nil {
				fmt.Printf(" | 💰 Cost: $%.4f", *m.TotalCostUSD)
			}
			fmt.Println()
		}
	}
}
