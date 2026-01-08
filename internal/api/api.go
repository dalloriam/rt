package api

import (
	"context"

	"github.com/dalloriam/rt/api"
)

// Runtime represents the runtime API exposed to runtime internals.
type Runtime interface {
	api.Runtime

	Options() *api.Options
	PublishEvent(ctx context.Context, event any)
}
