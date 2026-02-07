package rt

import (
	"github.com/dalloriam/rt/api"
	"github.com/dalloriam/rt/internal"
)

// Options reexports
type PubSubOptions = api.PubSubOptions
type Options = api.Options

type ProcessFn = api.ProcessFn
type ProcessState = api.ProcessState

type OnErrorStrategy = api.OnErrorStrategy

const (
	// OnErrorExit causes the process to exit if its internal function returns an error.
	OnErrorExit OnErrorStrategy = api.OnErrorExit

	// OnErrorRestart causes the process to be restarted if its internal function returns an error.
	OnErrorRestart OnErrorStrategy = api.OnErrorRestart
)

type SpawnOptions = api.SpawnOptions
type ProcessHandle = api.ProcessHandle
type Proc = api.Proc
type SubscriptionHandle = api.SubscriptionHandle
type PubSub = api.PubSub
type Runtime = api.Runtime

// New creates a new runtime with the given options.
func New(opts ...api.Options) api.Runtime {
	return internal.New(opts...)
}
