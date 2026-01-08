package proc

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/dalloriam/rt/api"
	"go.opentelemetry.io/otel/trace"
)

type Process struct {
	cancel  context.CancelFunc
	fn      api.ProcessFn
	manager *Manager
	opts    api.SpawnOptions
	running atomic.Bool
	pid     uint64
	span    trace.Span
	state   *api.ProcessState
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
		err := p.fn(p.state)

		p.manager.rt.PublishEvent(p.state.Ctx, processExitedEvent(p.pid, err))

		if err != nil {
			p.state.Log.Error("process failed", "error", err)

			switch p.opts.OnError {
			case api.OnErrorPanic:
				panic(err)
			case api.OnErrorExit:
				break outside
			case api.OnErrorRestart:
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
	p.manager.remove(p.pid)
	p.state.Log.Info("process stopped")
}

func (p *Process) IsRunning() bool {
	return p.running.Load()
}

// Stop stops the process.
func (p *Process) Stop() {
	if p.running.Load() {
		p.cancel()
		p.Wait()
	}
}

// Wait waits for the process to finish.
func (p *Process) Wait() {
	p.wg.Wait()
}
