package claudecode

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"testing"
	"time"
)

// TestContextCancellationLeak tests for resource leaks when context is cancelled without closing
func TestContextCancellationLeak(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	runtime.GC()
	initialGoroutines := runtime.NumGoroutine()
	t.Logf("Initial goroutines: %d", initialGoroutines)

	c, err := New(
		WithLogger(logger),
		WithMaxTurns(5),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())

	testSession, err := c.NewSession(ctx)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	msgChan, err := testSession.Receive(ctx)
	if err != nil {
		t.Fatalf("Failed to start receive: %v", err)
	}

	if err := testSession.Send(ctx, "Hello, please count from 1 to 10 slowly"); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Wait for some messages
	messageCount := 0
	timeout := time.After(3 * time.Second)

loop:
	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				break loop
			}
			messageCount++
			t.Logf("Received message %d: %T", messageCount, msg)
			if messageCount >= 2 {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	// Cancel context WITHOUT closing session
	t.Log("Cancelling context without closing session...")
	cancel()

	// Wait a bit for goroutines to potentially exit
	time.Sleep(2 * time.Second)

	// Check goroutine count
	runtime.GC()
	afterCancelGoroutines := runtime.NumGoroutine()
	t.Logf("Goroutines after cancel: %d", afterCancelGoroutines)

	t.Log("Now closing session properly...")
	if err := testSession.Close(); err != nil {
		t.Errorf("Error closing session: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Final goroutine count
	runtime.GC()
	finalGoroutines := runtime.NumGoroutine()
	t.Logf("Final goroutines: %d", finalGoroutines)

	// Check for leaks
	if afterCancelGoroutines > initialGoroutines+3 {
		t.Errorf("Potential goroutine leak after context cancel: started with %d, had %d after cancel",
			initialGoroutines, afterCancelGoroutines)
	}

	if finalGoroutines > initialGoroutines+1 {
		t.Errorf("Goroutine leak after close: started with %d, ended with %d",
			initialGoroutines, finalGoroutines)
	}
}

// TestProperContextHandling tests the recommended pattern with defer Close()
func TestProperContextHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	c, err := New(
		WithLogger(logger),
		WithMaxTurns(3),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testSession, err := c.NewSession(ctx)
	if err != nil {
		t.Fatalf("Failed to create testSession: %v", err)
	}
	defer testSession.Close()

	msgChan, err := testSession.Receive(ctx)
	if err != nil {
		t.Fatalf("Failed to start receive: %v", err)
	}

	if err := testSession.Send(ctx, "Say hello"); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Process some messages
	messageCount := 0
	for msg := range msgChan {
		messageCount++
		t.Logf("Received message: %T", msg)

		// Simulate context cancellation mid-stream
		if messageCount == 2 {
			cancel()
		}

		// Check if we should stop
		select {
		case <-ctx.Done():
			t.Log("Context cancelled, stopping message processing")
			return
		default:
		}
	}
}
