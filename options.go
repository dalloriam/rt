package rt

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

// Options contains options for the runtime.
type Options struct {
	// The pubsub configuration.
	PubSub PubSubOptions

	// The internal events configuration.
	// If nil, internal events are disabled.
	InternalEvents *InternalEventsOptions

	// Whether to enable opentelemetry.
	// If nil, opentelemetry is disabled.
	Telemetry *telemetry.Options
}
