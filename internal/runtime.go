package internal

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/dalloriam/rt/api"
	"github.com/dalloriam/rt/internal/proc"
	"github.com/dalloriam/rt/internal/pubsub"
	"github.com/dalloriam/rt/internal/sched"
	"github.com/dalloriam/rt/telemetry"
)

// runtime represents the main runtime environment.
type runtime struct {
	msg              *pubsub.PubSub
	log              *slog.Logger
	opts             api.Options
	proc             *proc.Manager
	sched            *sched.Scheduler
	telemetryCleanup func(context.Context) error
	tracer           trace.Tracer
}

func defaultOptions() api.Options {
	return api.Options{
		Log: api.LogOptions{
			Output: api.LogOutputStdout,
			Format: api.LogFormatText,
			Level:  "info",
		},
		PubSub: api.PubSubOptions{
			MaxBufferSize: 1024,
		},
		Telemetry: nil,
	}
}

// New returns a new runtime.
func New(opts ...api.Options) api.Runtime {

	options := defaultOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	log, err := newLogger(options.Log)
	if err != nil {
		panic(err)
	}

	log.Info("initializing runtime")

	rt := &runtime{
		log:    log,
		opts:   options,
		tracer: otel.Tracer("github.com/dalloriam/tools/gopkg/runtime"),
	}

	msg := pubsub.New(log, rt)
	rt.msg = msg

	if options.Telemetry != nil {
		var err error
		rt.telemetryCleanup, err = telemetry.Setup(context.Background(), *options.Telemetry)
		if err != nil {
			panic(err)
		}
		log.Info("OTel is enabled")
	}

	rt.proc = proc.NewManager(rt, log, rt.tracer)

	sched := sched.New(rt, log)
	rt.sched = sched

	log.Info("runtime ready")

	return rt
}

func (r *runtime) Proc() api.Proc {
	return r.proc
}

func (r *runtime) PubSub() api.PubSub {
	return r.msg
}

func (r *runtime) Scheduler() api.Scheduler {
	return r.sched
}

func (r *runtime) BlockUntilSignal() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		done <- true
	}()

	<-done
}

// Close closes the runtime, stopping all running processes.
func (r *runtime) Close() {
	r.log.Info("shutting down runtime")

	if err := r.proc.Close(); err != nil {
		r.log.Error("error while shutting down process manager", "error", err)
	}

	r.msg.Close()

	if r.telemetryCleanup != nil {
		if err := r.telemetryCleanup(context.Background()); err != nil {
			r.log.Error("error while shutting down telemetry", "error", err)
		}
	}

	r.log.Info("runtime shutdown complete")
}

func (r *runtime) Options() *api.Options {
	return &r.opts
}
