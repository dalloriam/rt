package proc

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/dalloriam/rt/api"
	"go.opentelemetry.io/otel/trace"
)

type process struct {
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

// PID returns the process ID.
func (p *process) PID() uint64 {
	return p.pid
}

// IsRunning returns true if the process is running.
func (p *process) IsRunning() bool {
	return p.running.Load()
}

// Stop stops the process.
func (p *process) Stop() {
	if p.running.Load() {
		p.cancel()
		p.Wait()
	}
}

// Wait waits for the process to finish.
func (p *process) Wait() {
	p.wg.Wait()
}

func (p *process) run() {
	defer p.wg.Done()

	if p.span != nil {
		defer p.span.End()
	}

	defer func() {
		p.running.Store(false)
		p.manager.unregister(p.pid)
		p.state.Log.Info("process stopped")
	}()

	p.running.Store(true)
	p.state.Log.Info("process started")

	for {
		err := p.fn(p.state)

		p.manager.rt.PublishEvent(p.state.Ctx, processExitedEvent(p.pid, err))

		if err != nil {
			p.state.Log.Error("process failed", "error", err)

			switch p.opts.OnError {
			case api.OnErrorExit:
				p.state.Log.Error("exiting process due to error")
				return
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
}
