package api

import (
	"github.com/dalloriam/rt/api"
)

// Runtime represents the runtime API exposed to runtime internals.
type Runtime interface {
	api.Runtime

	Options() *api.Options
}
