package proc_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	goruntime "runtime"

	"github.com/dalloriam/rt"
	"github.com/dalloriam/rt/api"
)

type testProcess struct {
	counter    int32
	ShouldFail bool
	StartCount atomic.Int32
}

func (p *testProcess) Run(state *api.ProcessState) error {
	p.StartCount.Add(1)
	if p.ShouldFail {
		return errors.New("some error")
	}

	for {
		select {
		case <-state.Ctx.Done():
			return nil

		default:
			p.counter++
		}
	}
}

func TestRuntimeProcSpawn_StartStop(t *testing.T) {
	r := rt.New()
	defer r.Close()

	p := &testProcess{}

	proc := r.Proc().Spawn(context.Background(), p.Run, api.SpawnOptions{OnError: api.OnErrorExit})

	time.Sleep(100 * time.Millisecond)
	if !proc.IsRunning() {
		t.Fatal("Process did not start")
	}
	proc.Stop()

	if p.counter == 0 {
		t.Fatal("Process did not run")
	}
}

func TestRuntimeShutdown_ProcCleanup(t *testing.T) {
	r := rt.New()

	p := &testProcess{}

	proc := r.Proc().Spawn(context.Background(), p.Run, api.SpawnOptions{OnError: api.OnErrorExit})

	time.Sleep(100 * time.Millisecond)
	if !proc.IsRunning() {
		t.Fatal("Process did not start")
	}

	r.Close()

	if proc.IsRunning() {
		t.Fatal("Process did not stop")
	}
}

func TestRuntimeProcSpawn_FailOnError(t *testing.T) {
	r := rt.New()
	defer r.Close()

	p := &testProcess{ShouldFail: true}

	proc := r.Proc().Spawn(context.Background(), p.Run, api.SpawnOptions{OnError: api.OnErrorExit})

	time.Sleep(100 * time.Millisecond)
	if proc.IsRunning() {
		t.Fatal("Process did not stop on error")
	}
}

func TestManager_RestartOnError(t *testing.T) {
	r := rt.New()
	defer r.Close()

	p := &testProcess{ShouldFail: true}

	proc := r.Proc().Spawn(context.Background(), p.Run, api.SpawnOptions{OnError: api.OnErrorRestart})

	time.Sleep(100 * time.Millisecond)
	if !proc.IsRunning() {
		t.Fatal("Process did not restart on error")
	}
}

func TestManager_ProcessWait(t *testing.T) {
	r := rt.New()
	defer r.Close()

	p := &testProcess{}

	proc := r.Proc().Spawn(context.Background(), p.Run, api.SpawnOptions{OnError: api.OnErrorExit})

	time.Sleep(100 * time.Millisecond)
	if !proc.IsRunning() {
		t.Fatal("Process did not start")
	}

	const nbWaiters = 3

	var wg sync.WaitGroup
	wg.Add(nbWaiters)

	for range nbWaiters {
		go func() {
			defer wg.Done()
			proc.Wait()
		}()
	}

	proc.Stop()
	wg.Wait() // ensure that stopping the process unblocks any waiters
}

// Reproduces issue #3 from REVIEW_REPORT.md:
// Stop() can be a no-op if called before the spawned goroutine marks itself running.
func TestProcess_StopImmediatelyAfterSpawn_DoesStop(t *testing.T) {
	prev := goruntime.GOMAXPROCS(1)
	defer goruntime.GOMAXPROCS(prev)

	r := rt.New(api.Options{
		Log: api.LogOptions{
			Output: api.LogOutputNone,
			Format: api.LogFormatText,
			Level:  "info",
		},
		PubSub: api.PubSubOptions{
			MaxBufferSize: 1024,
		},
	})
	defer r.Close()

	enteredWithLiveContext := make(chan struct{}, 1)

	proc := r.Proc().Spawn(context.Background(), func(state *api.ProcessState) error {
		if state.Ctx.Err() == nil {
			select {
			case enteredWithLiveContext <- struct{}{}:
			default:
			}
		}

		<-state.Ctx.Done()
		return nil
	}, api.SpawnOptions{OnError: api.OnErrorExit})

	// Intended to stop the process, but can race before running=true is set.
	proc.Stop()

	// Let the spawned goroutine run after Stop() returned.
	goruntime.Gosched()

	select {
	case <-enteredWithLiveContext:
		// Cleanup for this failing repro path.
		proc.Stop()
		t.Fatal("process entered function with live context after immediate Stop()")
	case <-time.After(25 * time.Millisecond):
	}
}
