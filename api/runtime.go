package api

import (
	"context"
	"log/slog"
)

type ProcessFn func(state *ProcessState) error

type ProcessState struct {
	Ctx context.Context
	Log *slog.Logger
	Rt  Runtime
}

type OnErrorStrategy int

const (
	// OnErrorPanic causes the process to panic if its internal function returns an error.
	OnErrorPanic OnErrorStrategy = iota

	// OnErrorExit causes the process to exit if its internal function returns an error.
	OnErrorExit

	// OnErrorRestart causes the process to be restarted if its internal function returns an error.
	OnErrorRestart
)

type SpawnOptions struct {
	// The error handling strategy for the process.
	OnError OnErrorStrategy

	// The name of the process.
	Name string

	// Whether to create a telemetry span for the process.
	Instrument bool
}

type ProcessHandle interface {
	IsRunning() bool
	PID() uint64
	Stop()
	Wait()
}

type Proc interface {
	Spawn(ctx context.Context, fn ProcessFn, opts SpawnOptions) ProcessHandle
}

type SubscriptionHandle interface {
	Data() <-chan any
	Close()
}

type PubSub interface {
	CreateTopic(name string, bufferSize uint32) error
	Subscribe(topicName, subscriptionName string) (SubscriptionHandle, error)
	Publish(ctx context.Context, topicName string, payload any) error
	Close()
}

type Runtime interface {
	BlockUntilSignal()
	Close()

	Proc() Proc
	PubSub() PubSub
}
