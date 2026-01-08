package internal

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dalloriam/rt/api"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slog.Level
	}{
		{
			name:     "debug level",
			input:    "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "info level",
			input:    "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "warn level",
			input:    "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "warning level",
			input:    "warning",
			expected: slog.LevelWarn,
		},
		{
			name:     "error level",
			input:    "error",
			expected: slog.LevelError,
		},
		{
			name:     "fatal level",
			input:    "fatal",
			expected: slog.LevelError,
		},
		{
			name:     "unknown level defaults to info",
			input:    "unknown",
			expected: slog.LevelInfo,
		},
		{
			name:     "empty string defaults to info",
			input:    "",
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLogLevel(tt.input)
			if got != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetLoggerOutput(t *testing.T) {
	t.Run("stdout output", func(t *testing.T) {
		output, err := getLoggerOutput(api.LogOutputStdout, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output != os.Stdout {
			t.Errorf("expected os.Stdout, got %v", output)
		}
	})

	t.Run("stderr output", func(t *testing.T) {
		output, err := getLoggerOutput(api.LogOutputStderr, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output != os.Stderr {
			t.Errorf("expected os.Stderr, got %v", output)
		}
	})

	t.Run("none output", func(t *testing.T) {
		output, err := getLoggerOutput(api.LogOutputNone, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output != io.Discard {
			t.Errorf("expected io.Discard, got %v", output)
		}
	})

	t.Run("file output", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "test.log")

		output, err := getLoggerOutput(api.LogOutputFile, logPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output == nil {
			t.Fatal("expected non-nil output")
		}

		// Verify file was created
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Errorf("log file was not created at %s", logPath)
		}

		// Clean up by closing the file
		if f, ok := output.(*os.File); ok {
			f.Close()
		}
	})

	t.Run("file output with invalid path", func(t *testing.T) {
		_, err := getLoggerOutput(api.LogOutputFile, "/invalid/path/that/does/not/exist/test.log")
		if err == nil {
			t.Error("expected error for invalid file path, got nil")
		}
	})

	t.Run("unknown output panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for unknown log output, but did not panic")
			}
		}()
		getLoggerOutput(api.LogOutput("unknown"), "")
	})
}

func TestNewLogger(t *testing.T) {
	t.Run("text format with info level", func(t *testing.T) {
		opts := api.LogOptions{
			Output: api.LogOutputStdout,
			Format: api.LogFormatText,
			Level:  "info",
		}

		// We can't directly test with stdout, so we'll test the logger creation
		logger, err := newLogger(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
	})

	t.Run("json format with debug level", func(t *testing.T) {
		opts := api.LogOptions{
			Output: api.LogOutputStdout,
			Format: api.LogFormatJSON,
			Level:  "debug",
		}

		logger, err := newLogger(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
	})

	t.Run("file output", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "test.log")

		opts := api.LogOptions{
			Output:   api.LogOutputFile,
			Format:   api.LogFormatText,
			Level:    "info",
			FilePath: logPath,
		}

		logger, err := newLogger(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}

		// Verify file was created
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Errorf("log file was not created at %s", logPath)
		}
	})

	t.Run("none output", func(t *testing.T) {
		opts := api.LogOptions{
			Output: api.LogOutputNone,
			Format: api.LogFormatText,
			Level:  "info",
		}

		logger, err := newLogger(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if logger == nil {
			t.Fatal("expected non-nil logger")
		}
	})
}

func TestLoggerIntegration(t *testing.T) {
	t.Run("logs are written to file in text format", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "test.log")

		opts := api.LogOptions{
			Output:   api.LogOutputFile,
			Format:   api.LogFormatText,
			Level:    "info",
			FilePath: logPath,
		}

		logger, err := newLogger(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Write some log messages
		logger.Info("test info message", "key", "value")
		logger.Debug("test debug message") // Should not appear (level is info)
		logger.Warn("test warning message")
		logger.Error("test error message")

		// Give the logger time to flush
		time.Sleep(10 * time.Millisecond)

		// Read the log file
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		logContent := string(content)

		// Verify expected messages are present
		if !strings.Contains(logContent, "test info message") {
			t.Error("expected info message not found in log")
		}
		if !strings.Contains(logContent, "test warning message") {
			t.Error("expected warning message not found in log")
		}
		if !strings.Contains(logContent, "test error message") {
			t.Error("expected error message not found in log")
		}

		// Verify debug message is not present (level is info)
		if strings.Contains(logContent, "test debug message") {
			t.Error("debug message should not be present when level is info")
		}
	})

	t.Run("logs are written to file in json format", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "test.log")

		opts := api.LogOptions{
			Output:   api.LogOutputFile,
			Format:   api.LogFormatJSON,
			Level:    "debug",
			FilePath: logPath,
		}

		logger, err := newLogger(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Write some log messages
		logger.Debug("test debug message", "key", "value")
		logger.Info("test info message")

		// Give the logger time to flush
		time.Sleep(10 * time.Millisecond)

		// Read the log file
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		logContent := string(content)

		// Verify it's valid JSON by parsing each line
		lines := strings.Split(strings.TrimSpace(logContent), "\n")
		if len(lines) != 2 {
			t.Errorf("expected 2 log lines, got %d", len(lines))
		}

		for i, line := range lines {
			var logEntry map[string]any
			if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
				t.Errorf("line %d is not valid JSON: %v", i+1, err)
			}

			// Verify required fields are present
			if _, ok := logEntry["time"]; !ok {
				t.Errorf("line %d missing 'time' field", i+1)
			}
			if _, ok := logEntry["level"]; !ok {
				t.Errorf("line %d missing 'level' field", i+1)
			}
			if _, ok := logEntry["msg"]; !ok {
				t.Errorf("line %d missing 'msg' field", i+1)
			}
		}
	})

	t.Run("debug level allows all messages", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "test.log")

		opts := api.LogOptions{
			Output:   api.LogOutputFile,
			Format:   api.LogFormatText,
			Level:    "debug",
			FilePath: logPath,
		}

		logger, err := newLogger(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Write messages at all levels
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")

		// Give the logger time to flush
		time.Sleep(10 * time.Millisecond)

		// Read the log file
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		logContent := string(content)

		// All messages should be present
		if !strings.Contains(logContent, "debug message") {
			t.Error("debug message not found")
		}
		if !strings.Contains(logContent, "info message") {
			t.Error("info message not found")
		}
		if !strings.Contains(logContent, "warn message") {
			t.Error("warn message not found")
		}
		if !strings.Contains(logContent, "error message") {
			t.Error("error message not found")
		}
	})

	t.Run("error level filters lower level messages", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "test.log")

		opts := api.LogOptions{
			Output:   api.LogOutputFile,
			Format:   api.LogFormatText,
			Level:    "error",
			FilePath: logPath,
		}

		logger, err := newLogger(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Write messages at all levels
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")

		// Give the logger time to flush
		time.Sleep(10 * time.Millisecond)

		// Read the log file
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		logContent := string(content)

		// Only error message should be present
		if strings.Contains(logContent, "debug message") {
			t.Error("debug message should be filtered")
		}
		if strings.Contains(logContent, "info message") {
			t.Error("info message should be filtered")
		}
		if strings.Contains(logContent, "warn message") {
			t.Error("warn message should be filtered")
		}
		if !strings.Contains(logContent, "error message") {
			t.Error("error message not found")
		}
	})

	t.Run("none output produces no logs", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "test.log")

		opts := api.LogOptions{
			Output:   api.LogOutputNone,
			Format:   api.LogFormatText,
			Level:    "debug",
			FilePath: logPath, // This should be ignored
		}

		logger, err := newLogger(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Write messages
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Error("error message")

		// Give the logger time to flush
		time.Sleep(10 * time.Millisecond)

		// Verify no file was created at the specified path
		if _, err := os.Stat(logPath); err == nil {
			t.Error("log file should not be created when output is none")
		}
	})
}

func TestRuntimeWithLogOptions(t *testing.T) {
	t.Run("runtime with file logging", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "runtime.log")

		opts := api.Options{
			Log: api.LogOptions{
				Output:   api.LogOutputFile,
				Format:   api.LogFormatText,
				Level:    "info",
				FilePath: logPath,
			},
		}

		rt := New(opts)
		if rt == nil {
			t.Fatal("expected non-nil runtime")
		}

		// Create a simple process to generate more logs
		testFn := func(state *api.ProcessState) error {
			state.Log.Info("process started")
			<-state.Ctx.Done()
			return nil
		}

		proc := rt.Proc().Spawn(context.Background(), testFn, api.SpawnOptions{Name: "test-proc"})
		time.Sleep(50 * time.Millisecond)
		proc.Stop()

		rt.Close()

		// Give the logger time to flush
		time.Sleep(10 * time.Millisecond)

		// Verify log file exists and contains runtime messages
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		logContent := string(content)

		// Verify runtime initialization messages
		if !strings.Contains(logContent, "initializing runtime") {
			t.Error("expected runtime initialization message not found")
		}
		if !strings.Contains(logContent, "runtime ready") {
			t.Error("expected runtime ready message not found")
		}
		if !strings.Contains(logContent, "shutting down runtime") {
			t.Error("expected runtime shutdown message not found")
		}
		if !strings.Contains(logContent, "process started") {
			t.Error("expected process log message not found")
		}
	})

	t.Run("runtime with none logging", func(t *testing.T) {
		opts := api.Options{
			Log: api.LogOptions{
				Output: api.LogOutputNone,
				Format: api.LogFormatText,
				Level:  "info",
			},
		}

		rt := New(opts)
		if rt == nil {
			t.Fatal("expected non-nil runtime")
		}

		rt.Close()
		// Should not panic or cause issues
	})

	t.Run("runtime with default options", func(t *testing.T) {
		rt := New()
		if rt == nil {
			t.Fatal("expected non-nil runtime")
		}

		// Default options should work without errors
		rt.Close()
	})

	t.Run("runtime with json logging", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "runtime.log")

		opts := api.Options{
			Log: api.LogOptions{
				Output:   api.LogOutputFile,
				Format:   api.LogFormatJSON,
				Level:    "debug",
				FilePath: logPath,
			},
		}

		rt := New(opts)
		rt.Close()

		// Give the logger time to flush
		time.Sleep(10 * time.Millisecond)

		// Verify log file contains valid JSON
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		for i, line := range lines {
			var logEntry map[string]any
			if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
				t.Errorf("line %d is not valid JSON: %v\nLine: %s", i+1, err, line)
			}
		}
	})
}
