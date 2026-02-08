package sched

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dalloriam/rt/api"
)

type fakeProcessHandle struct{}

func (f *fakeProcessHandle) IsRunning() bool { return false }
func (f *fakeProcessHandle) PID() uint64     { return 0 }
func (f *fakeProcessHandle) Stop()           {}
func (f *fakeProcessHandle) Wait()           {}

type fakeProc struct {
	count   atomic.Int64
	spawned chan struct{}
}

func newFakeProc() *fakeProc {
	return &fakeProc{
		spawned: make(chan struct{}, 256),
	}
}

func (p *fakeProc) Spawn(context.Context, api.ProcessFn, api.SpawnOptions) api.ProcessHandle {
	p.count.Add(1)
	select {
	case p.spawned <- struct{}{}:
	default:
	}
	return &fakeProcessHandle{}
}

func (p *fakeProc) Count() int64 {
	return p.count.Load()
}

type fakeRuntime struct {
	proc *fakeProc
	opts api.Options
}

func newFakeRuntime() *fakeRuntime {
	return &fakeRuntime{
		proc: newFakeProc(),
		opts: api.Options{
			PubSub: api.PubSubOptions{
				MaxBufferSize: 1024,
			},
		},
	}
}

func (r *fakeRuntime) BlockUntilSignal()        {}
func (r *fakeRuntime) Close()                   {}
func (r *fakeRuntime) Proc() api.Proc           { return r.proc }
func (r *fakeRuntime) PubSub() api.PubSub       { return nil }
func (r *fakeRuntime) Scheduler() api.Scheduler { return nil }
func (r *fakeRuntime) Options() *api.Options    { return &r.opts }

func TestSchedule_IgnoresNonPositiveInterval(t *testing.T) {
	rt := newFakeRuntime()
	s := New(rt, nil)
	defer s.Close()

	s.Schedule(func(*api.ProcessState) error { return nil }, api.SpawnOptions{}, 0)
	s.Schedule(func(*api.ProcessState) error { return nil }, api.SpawnOptions{}, -5*time.Millisecond)

	time.Sleep(30 * time.Millisecond)

	if got := rt.proc.Count(); got != 0 {
		t.Fatalf("expected no spawns for invalid intervals, got %d", got)
	}
}

func TestSchedule_UsesTickerAndStopsOnClose(t *testing.T) {
	rt := newFakeRuntime()
	s := New(rt, nil)

	s.Schedule(func(*api.ProcessState) error { return nil }, api.SpawnOptions{}, 5*time.Millisecond)

	waitForSpawns(t, rt.proc, 2, 250*time.Millisecond)

	s.Close()
	atClose := rt.proc.Count()
	time.Sleep(25 * time.Millisecond)
	afterClose := rt.proc.Count()

	if afterClose != atClose {
		t.Fatalf("spawns continued after scheduler close: atClose=%d afterClose=%d", atClose, afterClose)
	}
}

func TestSchedule_CloseIsIdempotent(t *testing.T) {
	rt := newFakeRuntime()
	s := New(rt, nil)

	s.Schedule(func(*api.ProcessState) error { return nil }, api.SpawnOptions{}, 5*time.Millisecond)

	waitForSpawns(t, rt.proc, 1, 250*time.Millisecond)

	// Closing multiple times must not panic.
	s.Close()
	s.Close()
	s.Close()
}

func waitForSpawns(t *testing.T, p *fakeProc, want int, timeout time.Duration) {
	t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for i := 0; i < want; i++ {
		select {
		case <-p.spawned:
		case <-timer.C:
			t.Fatalf("timeout waiting for %d spawns (got %d)", want, i)
		}
	}
}
