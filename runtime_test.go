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

func TestRuntime_PublishDuringClose_DoesNotPanic(t *testing.T) {
	const (
		attempts   = 200
		publishers = 64
		topicName  = "repro-topic"
	)

	for attempt := 1; attempt <= attempts; attempt++ {
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
