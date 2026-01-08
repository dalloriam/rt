package proc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dalloriam/rt/api"
	privApi "github.com/dalloriam/rt/internal/api"
	"go.opentelemetry.io/otel/trace"
)

type Manager struct {
	log    *slog.Logger
	mtx    sync.Mutex
	pid    uint64
	proc   map[uint64]*Process
	rt     privApi.Runtime
	tracer trace.Tracer
}

func NewManager(rt privApi.Runtime, log *slog.Logger, tracer trace.Tracer) *Manager {
	return &Manager{
		log:    log.WithGroup("proc"),
		proc:   make(map[uint64]*Process),
		rt:     rt,
		tracer: tracer,
	}
}

func (m *Manager) Spawn(ctx context.Context, fn api.ProcessFn, opts api.SpawnOptions) api.ProcessHandle {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	var span trace.Span

	if opts.Instrument {
		spanName := opts.Name
		if spanName == "" {
			spanName = fmt.Sprintf("process %d", m.pid)
		}

		ctx, span = m.tracer.Start(ctx, spanName)
	}

	ctx, cancel := context.WithCancel(ctx)

	log := m.log.With("pid", m.pid)
	if opts.Name != "" {
		log = log.With("name", opts.Name)
	}

	state := &api.ProcessState{
		Ctx: ctx,
		Log: log,
		Rt:  m.rt,
	}

	proc := &Process{
		cancel:  cancel,
		fn:      fn,
		manager: m,
		opts:    opts,
		pid:     m.pid,
		span:    span,
		state:   state,
		wg:      sync.WaitGroup{},
	}

	m.proc[m.pid] = proc
	m.pid++

	proc.wg.Add(1)
	go proc.run()

	m.rt.PublishEvent(ctx, processSpawnedEvent(proc.PID()))

	return proc
}

func (m *Manager) remove(pid uint64) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	delete(m.proc, pid)
}

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
