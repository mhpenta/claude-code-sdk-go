package claudecode

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

// TestSubprocessExitHandling tests that the subprocess transport handles Claude Code exits without panicking
func TestSubprocessExitHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	tests := []struct {
		name     string
		prompt   string
		maxTurns int
	}{
		{
			name:     "SimpleExit",
			prompt:   "Say 'test complete' and nothing else",
			maxTurns: 1,
		},
		{
			name:     "CodeReviewExit",
			prompt:   "Review this code for nil pointer issues: func test() { var p *int; println(*p) }. Be brief.",
			maxTurns: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				Logger:   logger,
				MaxTurns: tt.maxTurns,
			}

			transport := NewOneShotTransport(opts, tt.prompt)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Connect
			if err := transport.Connect(ctx); err != nil {
				t.Fatalf("Failed to connect: %v", err)
			}

			// Start receiving
			msgChan, err := transport.Receive(ctx)
			if err != nil {
				t.Fatalf("Failed to start receive: %v", err)
			}

			// Process messages
			messageCount := 0
			for range msgChan {
				messageCount++
			}

			t.Logf("Received %d messages", messageCount)

			// Small delay to let any pending goroutines finish
			time.Sleep(100 * time.Millisecond)

			// Close transport
			if err := transport.Close(); err != nil {
				t.Errorf("Error closing transport: %v", err)
			}

			// If we get here without panic, the test passed
			t.Logf("Test completed successfully without panic")
		})
	}
}

// TestSubprocessEarlyClose tests closing the transport while it's still processing
func TestSubprocessEarlyClose(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	opts := &Options{
		Logger:   logger,
		MaxTurns: 5,
	}

	transport := NewOneShotTransport(opts, "Count from 1 to 100")

	ctx := context.Background()

	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := transport.Receive(ctx)
	if err != nil {
		t.Fatalf("Failed to start receive: %v", err)
	}

	// Start processing in a goroutine
	done := make(chan bool)
	go func() {
		for range msgChan {
			// Just consume messages
		}
		done <- true
	}()

	// Close early after a short delay
	time.Sleep(1 * time.Second)
	t.Log("Closing transport early...")
	
	if err := transport.Close(); err != nil {
		t.Errorf("Error closing transport: %v", err)
	}

	// Wait for message processing to complete
	select {
	case <-done:
		t.Log("Message processing completed")
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for message processing to complete")
	}
}

// TestSubprocessNilLogger tests that nil logger doesn't cause panic
func TestSubprocessNilLogger(t *testing.T) {
	opts := &Options{
		Logger:   nil, // Explicitly nil logger
		MaxTurns: 1,
	}

	transport := NewOneShotTransport(opts, "Say hello")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := transport.Receive(ctx)
	if err != nil {
		t.Fatalf("Failed to start receive: %v", err)
	}

	// Process messages
	for range msgChan {
		// Just consume
	}

	// Should not panic even with nil logger
	if err := transport.Close(); err != nil {
		t.Errorf("Error closing transport: %v", err)
	}

	t.Log("Nil logger test completed without panic")
}