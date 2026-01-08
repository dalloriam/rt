package internal

import (
	"io"
	"log/slog"
	"os"

	"github.com/dalloriam/rt/api"
)

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error", "fatal":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getLoggerOutput(output api.LogOutput, filePath string) (io.Writer, error) {
	switch output {
	case api.LogOutputFile:
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
		panic("unknown log output")
	}
}

func newLogger(opts api.LogOptions) (*slog.Logger, error) {

	output, err := getLoggerOutput(opts.Output, opts.FilePath)
	if err != nil {
		return nil, err
	}

	handlerOpts := &slog.HandlerOptions{
		Level: parseLogLevel(opts.Level),
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
