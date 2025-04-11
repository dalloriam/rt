package rt_test

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

func doTest(t *testing.T, buffering int, subCount int, consumersPerSub int) {
	rt := rt.New()

	if err := rt.CreateTopic(TopicName, buffering); err != nil {
		t.Fatalf("failed to create topic: %v", err)
	}

	consumedPerSubscription := make([]*Results, subCount)
	for i := 0; i < subCount; i++ {
		consumedPerSubscription[i] = &Results{}
	}

	var wg sync.WaitGroup
	wg.Add(subCount * consumersPerSub)

	for i := 0; i < subCount; i++ {
		subName := fmt.Sprintf("sub-%d", i)

		for j := 0; j < consumersPerSub; j++ {
			ch, err := rt.Subscribe(TopicName, subName)
			defer ch.Close()
			if err != nil {
				panic(err)
			}
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

	for i := 0; i < 10; i++ {
		if err := rt.Publish(context.Background(), TopicName, fmt.Sprintf("bing%d", i)); err != nil {
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
	for _, topicBufferSize := range []int{0, 10, 100} {
		for _, subCount := range []int{1, 2, 100} {
			for _, consumersPerSub := range []int{1, 2, 100} {
				t.Run(fmt.Sprintf("Buf=%d,Sub=%d,Cons=%d", topicBufferSize, subCount, consumersPerSub), func(t *testing.T) {
					doTest(t, topicBufferSize, subCount, consumersPerSub)
				})
			}
		}
	}
}
