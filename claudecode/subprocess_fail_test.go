package claudecode

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

// TestSubprocessFailToStart tests the case where the process doesn't even start
func TestSubprocessFailToStart(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	tests := []struct {
		name    string
		options *Options
	}{
		{
			name: "InvalidCLIPath",
			options: &Options{
				Logger:  logger,
				CLIPath: "/does/not/exist/claude",
			},
		},
		{
			name: "InvalidWorkingDirectory",
			options: &Options{
				Logger:           logger,
				WorkingDirectory: "/does/not/exist/dir",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewOneShotTransport(tt.options, "test")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// This should fail
			err := transport.Connect(ctx)
			if err == nil {
				t.Fatal("Expected Connect to fail, but it succeeded")
			}
			t.Logf("Connect failed as expected: %v", err)

			// Make sure we don't panic on cleanup
			err = transport.Close()
			if err != nil {
				t.Logf("Close returned error: %v", err)
			}

			// Give time for any goroutines to finish
			time.Sleep(100 * time.Millisecond)

			t.Log("Test completed without panic")
		})
	}
}

// TestSubprocessPanicScenarios tests various scenarios that might cause panics
func TestSubprocessPanicScenarios(t *testing.T) {
	// Test with nil logger
	t.Run("NilLogger", func(t *testing.T) {
		opts := &Options{
			Logger:  nil, // Explicitly nil
			CLIPath: "/does/not/exist/claude",
		}

		transport := NewOneShotTransport(opts, "test")
		ctx := context.Background()

		// Should handle nil logger gracefully
		_ = transport.Connect(ctx)
		_ = transport.Close()

		t.Log("Nil logger test completed without panic")
	})

	// Test rapid close after connect
	t.Run("RapidClose", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		opts := &Options{
			Logger: logger,
		}

		transport := NewOneShotTransport(opts, "test")
		ctx := context.Background()

		if err := transport.Connect(ctx); err != nil {
			t.Logf("Connect failed: %v", err)
			return
		}

		// Immediately close
		_ = transport.Close()

		t.Log("Rapid close test completed without panic")
	})

	// Test context cancellation during receive
	t.Run("ContextCancelDuringReceive", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		opts := &Options{
			Logger: logger,
		}

		transport := NewOneShotTransport(opts, "say hello")
		ctx, cancel := context.WithCancel(context.Background())

		if err := transport.Connect(ctx); err != nil {
			t.Logf("Connect failed: %v", err)
			return
		}

		_, err := transport.Receive(ctx)
		if err != nil {
			t.Logf("Receive failed: %v", err)
			return
		}

		// Cancel context immediately
		cancel()
		
		// Wait a bit
		time.Sleep(100 * time.Millisecond)

		// Close transport
		_ = transport.Close()

		t.Log("Context cancel test completed without panic")
	})
}