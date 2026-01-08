package pubsub_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/dalloriam/rt"
)

const TopicName = "test-topic"

type Results struct {
	sync.Mutex
	Data []any
}

func doTest(t *testing.T, buffering uint32, subCount int, consumersPerSub int) {
	rt := rt.New()

	if err := rt.PubSub().CreateTopic(TopicName, buffering); err != nil {
		t.Fatalf("failed to create topic: %v", err)
	}

	consumedPerSubscription := make([]*Results, subCount)
	for i := range subCount {
		consumedPerSubscription[i] = &Results{}
	}

	var wg sync.WaitGroup
	wg.Add(subCount * consumersPerSub)

	for i := range subCount {
		subName := fmt.Sprintf("sub-%d", i)

		for range consumersPerSub {
			ch, err := rt.PubSub().Subscribe(TopicName, subName)
			if err != nil {
				panic(err)
			}
			defer ch.Close()

			go func() {
				for msg := range ch.Data() {
					consumedPerSubscription[i].Lock()
					consumedPerSubscription[i].Data = append(consumedPerSubscription[i].Data, msg)
					consumedPerSubscription[i].Unlock()
				}

				wg.Done()
			}()
		}

	}

	for i := range 10 {
		if err := rt.PubSub().Publish(context.Background(), TopicName, fmt.Sprintf("bing%d", i)); err != nil {
			t.Fatalf("failed to publish: %v", err)
		}
	}

	rt.Close()
	wg.Wait()

	for i, consumed := range consumedPerSubscription {
		if len(consumed.Data) != 10 {
			t.Fatalf("expected 10 messages for subscription %d, got %d", i, len(consumed.Data))
		}
	}
}

func TestEventDispatch_PubSub(t *testing.T) {
	for _, topicBufferSize := range []uint32{0, 10, 100} {
		for _, subCount := range []int{1, 2, 100} {
			for _, consumersPerSub := range []int{1, 2, 100} {
				t.Run(fmt.Sprintf("Buf=%d,Sub=%d,Cons=%d", topicBufferSize, subCount, consumersPerSub), func(t *testing.T) {
					doTest(t, topicBufferSize, subCount, consumersPerSub)
				})
			}
		}
	}
}
