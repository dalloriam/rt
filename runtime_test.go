package rt_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	rt "github.com/dalloriam/rt"
	"github.com/dalloriam/rt/api"
)

func TestNew_InvalidLogLevel_ReturnsError(t *testing.T) {
	_, err := rt.New(api.Options{
		Log: api.LogOptions{
			Output: api.LogOutputStdout,
			Format: api.LogFormatText,
			Level:  "bogus",
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid log level, got nil")
	}
}

func TestNew_InvalidLogOutput_ReturnsError(t *testing.T) {
	_, err := rt.New(api.Options{
		Log: api.LogOptions{
			Output: api.LogOutput("invalid"),
			Format: api.LogFormatText,
			Level:  "info",
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid log output, got nil")
	}
}

func TestMustNew_PanicsOnError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected MustNew to panic on invalid options")
		}
	}()
	rt.MustNew(api.Options{
		Log: api.LogOptions{
			Output: api.LogOutput("invalid"),
			Level:  "info",
		},
	})
}

func TestRuntime_PublishDuringClose_DoesNotPanic(t *testing.T) {
	const (
		attempts   = 200
		publishers = 64
		topicName  = "repro-topic"
	)

	for attempt := 1; attempt <= attempts; attempt++ {
		r := rt.MustNew(api.Options{
			Log: api.LogOptions{
				Output: api.LogOutputNone,
				Format: api.LogFormatText,
				Level:  "info",
			},
			PubSub: api.PubSubOptions{
				MaxBufferSize: 1024,
			},
		})

		if err := r.PubSub().CreateTopic(topicName, 0); err != nil {
			t.Fatalf("attempt %d: create topic: %v", attempt, err)
		}

		start := make(chan struct{})
		stop := make(chan struct{})
		var panicked atomic.Bool
		var panicOnce sync.Once
		var panicValue any

		var wg sync.WaitGroup
		for i := 0; i < publishers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start

				for {
					select {
					case <-stop:
						return
					default:
					}

					func() {
						defer func() {
							if recovered := recover(); recovered != nil {
								panicked.Store(true)
								panicOnce.Do(func() {
									panicValue = recovered
								})
							}
						}()
						_ = r.PubSub().Publish(context.Background(), topicName, "payload")
					}()
				}
			}()
		}

		close(start)
		time.Sleep(5 * time.Millisecond)
		r.Close()
		close(stop)
		wg.Wait()

		if panicked.Load() {
			t.Fatalf("attempt %d: panic while publishing during close: %v", attempt, panicValue)
		}
	}
}

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

func TestRuntimeCloseIsIdempotent(t *testing.T) {
	rt, err := rt.New()
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}

	// Closing multiple times must not panic.
	rt.Close()
	rt.Close()
	rt.Close()
}

// runtime.Close() must stop scheduler goroutines.
func TestRuntimeCloseStopsScheduler(t *testing.T) {
	rt, err := rt.New()
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
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
