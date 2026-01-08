package internal

import (
	"errors"
	"io"
	"log/slog"
	"os"

	"github.com/dalloriam/rt/api"
)

func parseLogLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error", "fatal":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, errors.New("invalid log level")
	}
}

func getLoggerOutput(output api.LogOutput, filePath string) (io.Writer, error) {
	switch output {
	case api.LogOutputFile:
		// FIXME: We're leaking file handles here every time a runtime is shut down (shouldn't happen very often in practice)
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		return f, nil
	case api.LogOutputStdout:
		return os.Stdout, nil
	case api.LogOutputStderr:
		return os.Stderr, nil
	case api.LogOutputNone:
		return io.Discard, nil
	default:
		return nil, errors.New("invalid log output option")
	}
}

func newLogger(opts api.LogOptions) (*slog.Logger, error) {
	output, err := getLoggerOutput(opts.Output, opts.FilePath)
	if err != nil {
		return nil, err
	}

	level, err := parseLogLevel(opts.Level)
	if err != nil {
		return nil, err
	}

	handlerOpts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	switch opts.Format {
	case api.LogFormatJSON:
		handler = slog.NewJSONHandler(output, handlerOpts)
	case api.LogFormatText:
		handler = slog.NewTextHandler(output, handlerOpts)
	default:
		handler = slog.NewTextHandler(output, handlerOpts)
	}

	logger := slog.New(handler)

	return logger, nil
}
