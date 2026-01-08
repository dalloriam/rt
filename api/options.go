package api

import "github.com/dalloriam/rt/telemetry"

// InternalEventsOptions contains options for internal events.
type InternalEventsOptions struct {
	// The topic to publish internal events to.
	Topic string

	// The buffer size for the internal events topic.
	BufferSize uint32
}

// PubSubOptions contains options for the pubsub system.
type PubSubOptions struct {
	MaxBufferSize uint32
}

type LogOutput string

const (
	// LogOutputStdout indicates that logs should be written to stdout.
	LogOutputStdout LogOutput = "stdout"

	// LogOutputStderr indicates that logs should be written to stderr.
	LogOutputStderr LogOutput = "stderr"

	// LogOutputFile indicates that logs should be written to a file.
	LogOutputFile LogOutput = "file"

	// LogOutputNone indicates that no logs should be written.
	LogOutputNone LogOutput = "none"
)

type LogFormat string

const (
	// LogFormatText indicates that logs should be written in text format.
	LogFormatText LogFormat = "text"

	// LogFormatJSON indicates that logs should be written in JSON format.
	LogFormatJSON LogFormat = "json"
)

// LogOptions contains options for logging.
type LogOptions struct {
	// The log output type.
	Output LogOutput

	// The log format.
	Format LogFormat

	// The log level.
	Level string

	// The log file path. Only used if Output is LogOutputFile.
	FilePath string
}

// Options contains options for the runtime.
type Options struct {
	// The logging configuration.
	Log LogOptions

	// The pubsub configuration.
	PubSub PubSubOptions

	// The internal events configuration.
	// If nil, internal events are disabled.
	InternalEvents *InternalEventsOptions

	// Whether to enable opentelemetry.
	// If nil, opentelemetry is disabled.
	Telemetry *telemetry.Options
}
