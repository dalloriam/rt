package proc_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
