package main

import (
	"context"
	"fmt"
	"log"
	
	"github.com/mhpenta/claude-code-sdk-go/claudecode"
)

func main() {
	// Test if allowed tools actually work
	client, err := claudecode.New(
		claudecode.WithAllowedTools("Read", "Grep"),
		claudecode.WithPermissionMode(claudecode.PermissionModeDefault),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	
	ctx := context.Background()
	
	// Ask Claude to use a tool
	messages, err := client.Query(ctx, "List the files in the current directory")
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	
	// Check what Claude responds
	for _, msg := range messages {
		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			fmt.Println("Claude's response:")
			for _, block := range m.Content {
				if block.Type == "text" && block.Text != nil {
					fmt.Println(*block.Text)
				}
				if block.Type == "tool_use" && block.Tool != nil {
					fmt.Printf("Tool used: %s\n", block.Tool.Name)
				}
			}
		case *claudecode.ResultMessage:
			fmt.Printf("\nCompleted in %dms\n", m.DurationMS)
		}
	}
}