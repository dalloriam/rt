package internal

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/dalloriam/rt/api"
)

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, msg string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}

	t.Fatalf("timeout waiting for condition: %s", msg)
}

// Regression test for issue #1 from REVIEW_REPORT.md:
// runtime.Close() must stop scheduler goroutines.
func TestRuntimeCloseStopsScheduler(t *testing.T) {
	rtAPI := New()
	rt, ok := rtAPI.(*runtime)
	if !ok {
		t.Fatalf("expected *runtime, got %T", rtAPI)
	}

	var runs atomic.Int64
	process := func(_ *api.ProcessState) error {
		runs.Add(1)
		return nil
	}

	rt.Scheduler().Schedule(process, api.SpawnOptions{OnError: api.OnErrorExit}, 5*time.Millisecond)

	waitForCondition(t, 500*time.Millisecond, func() bool {
		return runs.Load() > 0
	}, "scheduled process to run at least once")

	rt.Close()

	atClose := runs.Load()
	time.Sleep(50 * time.Millisecond)
	afterClose := runs.Load()

	if afterClose != atClose {
		t.Fatalf("scheduler continued after runtime.Close(): runs at close=%d after=%d", atClose, afterClose)
	}
}
