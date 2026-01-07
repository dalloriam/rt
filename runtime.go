package rt

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/dalloriam/rt/telemetry"
)

// Runtime represents the main runtime environment.
type Runtime struct {
	evt              *pubsubDispatch
	log              *slog.Logger
	opts             Options
	proc             *processManager
	sched            *scheduler
	telemetryCleanup func(context.Context) error
	tracer           trace.Tracer
}

func defaultOptions() Options {
	return Options{
		PubSub: PubSubOptions{
			MaxBufferSize: 1024,
		},
		InternalEvents:   nil,
		TelemetryOptions: nil,
	}
}

// New returns a new runtime.
func New(opts ...Options) *Runtime {

	options := defaultOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	// TODO: Improve logging.
	// The runtime should:
	// - have the option to emit logs to disk
	// - allow processes to _read_ the logs (allowing for a "debugger" process)
	log := slog.Default()

	log.Info("initializing runtime")

	rt := &Runtime{
		log:    log,
		opts:   options,
		tracer: otel.Tracer("github.com/dalloriam/tools/gopkg/runtime"),
	}

	evt := newPubSub(log, rt)

	if options.InternalEvents != nil {
		if err := evt.CreateTopic(options.InternalEvents.Topic, options.InternalEvents.BufferSize); err != nil {
			panic(err)
		}
	}

	rt.evt = evt

	if options.TelemetryOptions != nil {
		var err error
		rt.telemetryCleanup, err = telemetry.Setup(context.Background(), *options.TelemetryOptions)
		if err != nil {
			panic(err)
		}
		log.Info("OTel is enabled")
	}

	proc := newProcessManager(rt, log)
	rt.proc = proc

	sched := newScheduler(rt, log)
	rt.sched = sched

	log.Info("runtime ready")

	return rt
}

func (r *Runtime) tryPublishEvent(ctx context.Context, event any) {
	if r.opts.InternalEvents != nil {
		if err := r.evt.Publish(ctx, r.opts.InternalEvents.Topic, event); err != nil {
			r.log.Error("error while publishing internal event", "error", err)
		}
	}
}

// ProcSpawn spawns a new process and returns a process handle.
func (r *Runtime) ProcSpawn(ctx context.Context, fn ProcessFn, opts ProcessSpawnOptions) *Process {
	proc := r.proc.Add(ctx, fn, opts)

	r.tryPublishEvent(ctx, processSpawnedEvent(proc.PID()))

	return proc
}

func (r *Runtime) BlockUntilSignal() {
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
func (r *Runtime) Close() {
	r.tryPublishEvent(context.Background(), closeEvent())

	r.log.Info("shutting down runtime")

	if err := r.proc.Close(); err != nil {
		r.log.Error("error while shutting down process manager", "error", err)
	}

	r.evt.Close()

	if r.telemetryCleanup != nil {
		if err := r.telemetryCleanup(context.Background()); err != nil {
			r.log.Error("error while shutting down telemetry", "error", err)
		}
	}

	r.log.Info("runtime shutdown complete")
}

// CreateTopic creates a new topic with the given name and buffer size.
// Returns an error if the topic already exists or if the buffer size exceeds
// the maximum allowed size.
func (r *Runtime) CreateTopic(name string, bufferSize uint32) error {
	if bufferSize > r.opts.PubSub.MaxBufferSize {
		return ErrBufferSizeTooLarge
	}

	if name == "" {
		return ErrInvalidTopicName
	}

	err := r.evt.CreateTopic(name, bufferSize)

	r.tryPublishEvent(context.Background(), createTopicEvent(name, bufferSize, err))

	return err
}

// Publish publishes a message to a topic.
// Returns an error if the topic does not exist or if publishing fails.
func (r *Runtime) Publish(ctx context.Context, topic string, msg any) error {
	err := r.evt.Publish(ctx, topic, msg)
	r.tryPublishEvent(ctx, publishEvent(topic, msg, err))

	return err
}

// Subscribe subscribes to a topic.
// Returns a subscription handle or an error if the topic does not exist
func (r *Runtime) Subscribe(topic, subscription string) (*Subscription, error) {
	sub, err := r.evt.Subscribe(topic, subscription)

	r.tryPublishEvent(context.Background(), createSubscriptionEvent(topic, subscription, err))

	return sub, err
}

func (r *Runtime) Schedule(fn ProcessFn, opts ProcessSpawnOptions, interval time.Duration) {
	r.sched.Schedule(fn, opts, interval)
}
