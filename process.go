package rt

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type ProcessState struct {
	Ctx context.Context
	Log *slog.Logger
	Rt  *Runtime
	Sig <-chan any
}

type ProcessFn func(state *ProcessState) error

type Process struct {
	cancel  context.CancelFunc
	fn      ProcessFn
	manager *processManager
	opts    ProcessSpawnOptions
	running atomic.Bool
	pid     uint64
	span    trace.Span
	state   *ProcessState
	sig     chan any
	wg      sync.WaitGroup
}

func (p *Process) PID() uint64 {
	return p.pid
}

func (p *Process) run() {
	defer p.wg.Done()
	if p.span != nil {
		defer p.span.End()
	}
	p.running.Store(true)
	p.state.Log.Info("process started")

outside:
	for {
		if err := p.fn(p.state); err != nil {
			p.state.Log.Error("process failed", "error", err)

			switch p.opts.ErrorHandling {
			case ProcessErrorPanic:
				panic(err)

			case ProcessErrorExit:
				break outside

			case ProcessErrorRestart:
				p.state.Log.Info("restarting process")
			}
		} else {
			break
		}

		if p.state.Ctx.Err() != nil {
			// Context was cancelled.
			break
		}
	}
	p.running.Store(false)
	p.manager.Remove(p.pid)
	p.state.Log.Info("process stopped")
}

func (p *Process) IsRunning() bool {
	return p.running.Load()
}

// Signal sends a signal to the process.
func (p *Process) Signal(sig any) {
	select {
	case <-p.state.Ctx.Done():
		return
	case <-time.After(5 * time.Second):
		p.state.Log.Error("signal send timeout")
		return
	case p.sig <- sig:
	}
}

// Stop stops the process.
func (p *Process) Stop() {
	if p.running.Load() {
		p.cancel()
		p.wg.Wait()
		close(p.sig)
	}
}

type ProcessError int

const (
	ProcessErrorPanic ProcessError = iota
	ProcessErrorExit
	ProcessErrorRestart
)

type ProcessSpawnOptions struct {
	// The error handling strategy for the process.
	ErrorHandling ProcessError

	// The name of the process.
	Name string

	// Whether to create a telemetry span for the process.
	Instrument bool
}

type processManager struct {
	log  *slog.Logger
	mtx  sync.Mutex
	pid  uint64
	proc map[uint64]*Process
	rt   *Runtime
}

func newProcessManager(rt *Runtime, log *slog.Logger) *processManager {
	return &processManager{
		log:  log.WithGroup("proc"),
		proc: make(map[uint64]*Process),
		rt:   rt,
	}
}

func (m *processManager) Add(ctx context.Context, fn ProcessFn, opts ProcessSpawnOptions) *Process {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	var span trace.Span

	if opts.Instrument {
		spanName := opts.Name
		if spanName == "" {
			spanName = fmt.Sprintf("process %d", m.pid)
		}

		ctx, span = m.rt.tracer.Start(ctx, spanName)
	}

	ctx, cancel := context.WithCancel(ctx)
	sig := make(chan any)

	log := m.log.With("pid", m.pid)
	if opts.Name != "" {
		log = log.With("name", opts.Name)
	}

	state := &ProcessState{
		Ctx: ctx,
		Log: log,
		Rt:  m.rt,
		Sig: sig,
	}

	proc := &Process{
		cancel:  cancel,
		fn:      fn,
		manager: m,
		opts:    opts,
		pid:     m.pid,
		span:    span,
		state:   state,
		sig:     sig,
		wg:      sync.WaitGroup{},
	}

	m.proc[m.pid] = proc
	m.pid++

	proc.wg.Add(1)
	go proc.run()

	return proc
}

func (m *processManager) Remove(pid uint64) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if proc, ok := m.proc[pid]; ok {
		proc.Stop()
		delete(m.proc, pid)
	}
}

func (m *processManager) Close() error {
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
