package proc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/dalloriam/rt/api"
	privApi "github.com/dalloriam/rt/internal/api"
	"go.opentelemetry.io/otel/trace"
)

// Manager manages processes.
type Manager struct {
	log    *slog.Logger
	pid    atomic.Uint64
	rt     privApi.Runtime
	tracer trace.Tracer

	mtx  sync.Mutex
	proc map[uint64]*process
}

// NewManager creates a new process manager.
func NewManager(rt privApi.Runtime, log *slog.Logger, tracer trace.Tracer) *Manager {
	return &Manager{
		log:    log.WithGroup("proc"),
		pid:    atomic.Uint64{},
		rt:     rt,
		tracer: tracer,

		proc: make(map[uint64]*process),
	}
}

// Spawn spawns a new process with the given function and options.
func (m *Manager) Spawn(ctx context.Context, fn api.ProcessFn, opts api.SpawnOptions) api.ProcessHandle {
	var span trace.Span

	pid := m.pid.Add(1)

	if opts.Instrument {
		spanName := opts.Name
		if spanName == "" {
			spanName = fmt.Sprintf("process %d", pid)
		}

		ctx, span = m.tracer.Start(ctx, spanName)
	}

	ctx, cancel := context.WithCancel(ctx)

	log := m.log.With("pid", pid)
	if opts.Name != "" {
		log = log.With("name", opts.Name)
	}

	state := &api.ProcessState{
		Ctx: ctx,
		Log: log,
		Rt:  m.rt,
	}

	proc := &process{
		cancel:  cancel,
		fn:      fn,
		manager: m,
		opts:    opts,
		pid:     pid,
		span:    span,
		state:   state,
		wg:      sync.WaitGroup{},
	}

	m.register(pid, proc)

	proc.wg.Add(1)
	go proc.run()

	m.rt.PublishEvent(ctx, processSpawnedEvent(proc.PID()))

	return proc
}

// Close stops all running processes.
func (m *Manager) Close() error {
	var wg sync.WaitGroup
	m.mtx.Lock()

	// FIXME: Add a timeout in case a process doesn't stop.

	for _, proc := range m.proc {
		wg.Add(1)
		go func() {
			proc.Stop()
			wg.Done()
		}()
	}
	m.mtx.Unlock()

	wg.Wait()

	return nil
}

func (m *Manager) register(pid uint64, proc *process) {
	m.mtx.Lock()
	m.proc[pid] = proc
	m.mtx.Unlock()
}

func (m *Manager) unregister(pid uint64) {
	m.mtx.Lock()
	delete(m.proc, pid)
	m.mtx.Unlock()
}
