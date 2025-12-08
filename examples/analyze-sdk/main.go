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

	prompt := `Analyze the architecture of this Go SDK in the claudecode/ directory.
Focus on:
1. Design patterns used
2. Interface design
3. Strengths of the architecture

Be concise.`

	messages, err := client.Query(ctx, prompt)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	printResponse(messages)
	return nil
}

func reviewCodeQuality(client claudecode.Client) error {
	ctx := context.Background()

	prompt := `Review code quality in the claudecode/ directory.
Look for:
1. Go idioms and best practices
2. Error handling
3. Concurrency safety

Highlight good practices and concerns.`

	messages, err := client.Query(ctx, prompt)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	printResponse(messages)
	return nil
}

func suggestImprovements(client claudecode.Client) error {
	ctx := context.Background()

	prompt := `Suggest 3-5 improvements for this Go SDK that would:
1. Make it more robust
2. Improve developer experience
3. Better align with Go best practices

Briefly explain why each is important.`

	msgChan, err := client.QueryStream(ctx, prompt)
	if err != nil {
		return fmt.Errorf("query stream failed: %w", err)
	}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				if block.Type == "text" && block.Text != nil {
					fmt.Print(*block.Text)
				}
			}
		case *claudecode.ResultMessage:
			fmt.Printf("\n\nDuration: %dms", m.DurationMS)
			if m.TotalCostUSD != nil {
				fmt.Printf(" | Cost: $%.4f", *m.TotalCostUSD)
			}
			fmt.Println()
		}
	}
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
