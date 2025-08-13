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
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	fmt.Println("SDK Analysis")
	fmt.Println("============")
	fmt.Printf("Project root: %s\n\n", projectRoot)

	client, err := claudecode.New(
		claudecode.WithWorkingDirectory(projectRoot),
		claudecode.WithLogger(logger),
		claudecode.WithSystemPrompt("Focus on Go best practices and architecture patterns."),
		claudecode.WithPermissionMode(claudecode.PermissionModeDefault),
		claudecode.WithAddDirs(filepath.Join(projectRoot, "claudecode")),
	)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()

	fmt.Println("\nAnalyzing architecture...")
	if err := analyzeArchitecture(client); err != nil {
		log.Fatal("Architecture analysis failed:", err)
	}


	fmt.Println("\nReviewing code quality...")
	if err := reviewCodeQuality(client); err != nil {
		log.Fatal("Code quality review failed:", err)
	}


	fmt.Println("\nGenerating improvement suggestions...")
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
			fmt.Printf("\nDuration: %dms", m.DurationMS)
			if m.TotalCostUSD != nil {
				fmt.Printf(" | Cost: $%.4f", *m.TotalCostUSD)
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
			fmt.Printf("\n\nDuration: %dms", m.DurationMS)
			if m.TotalCostUSD != nil {
				fmt.Printf(" | Cost: $%.4f", *m.TotalCostUSD)
			}
			fmt.Println()
		}
	}
}
