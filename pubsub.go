package rt

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/trace"
)

type Subscription struct {
	name       string
	topic      *topic
	rx         chan any
	closed     atomic.Bool
	cancelChan chan struct{}
}

func newSubscription(name string, t *topic) *Subscription {
	return &Subscription{
		name:       name,
		topic:      t,
		rx:         make(chan any),
		cancelChan: make(chan struct{}),
	}
}

// Data returns the channel on which the subscription receives data.
func (s *Subscription) Data() <-chan any {
	return s.rx
}

// Close closes the subscription for every consumer.
// Safe to call multiple times per subscription
// (although all subscribers will stop receiving messages once called once).
func (s *Subscription) Close() {
	if s.closed.CompareAndSwap(false, true) {
		close(s.cancelChan)
		s.topic.Unsubscribe(s.name)
		close(s.rx)
	}
}

type topic struct {
	tracer trace.Tracer
	rx     chan any
	wg     sync.WaitGroup

	mtx           sync.Mutex
	subscriptions map[string]*Subscription

	// this channel gets closed when the first subscriber is added. this is so we dont start reading until there is a consumer
	emptySubChannel      chan struct{}
	atLeastOneSubCreated bool
}

func newTopic(bufferSize int, tracer trace.Tracer) *topic {
	t := &topic{
		tracer:          tracer,
		rx:              make(chan any, bufferSize),
		subscriptions:   make(map[string]*Subscription),
		emptySubChannel: make(chan struct{}),
	}

	t.wg.Add(1)
	go t.run()

	return t
}

func (b *topic) run() {
	var dispatchWg sync.WaitGroup

	// wait to get at least one subscriber
	select {
	case <-b.emptySubChannel:
	}

	for evt := range b.rx {
		b.mtx.Lock()
		for _, sub := range b.subscriptions {
			dispatchWg.Add(1)
			go func() {
				select {
				case <-sub.cancelChan:
					dispatchWg.Done()
					return
				case sub.rx <- evt:
					dispatchWg.Done()
					return
				}
			}()
		}
		dispatchWg.Wait()
		b.mtx.Unlock()
	}

	b.wg.Done()
}

func (b *topic) Subscribe(subName string) (*Subscription, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if sub, ok := b.subscriptions[subName]; ok {
		return sub, nil
	}

	source := newSubscription(subName, b)
	b.subscriptions[subName] = source

	if !b.atLeastOneSubCreated {
		close(b.emptySubChannel)
	}

	b.atLeastOneSubCreated = true

	return source, nil
}

func (b *topic) Unsubscribe(subName string) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	delete(b.subscriptions, subName)
}

func (b *topic) Publish(ctx context.Context, evt any) error {
	ctx, span := b.tracer.Start(ctx, "Topic.Publish")
	defer span.End()
	b.rx <- evt
	return nil
}

func (b *topic) Close() {
	b.mtx.Lock()
	close(b.rx)
	if !b.atLeastOneSubCreated {
		close(b.emptySubChannel)
	}
	b.mtx.Unlock()

	b.wg.Wait()

	var subs []*Subscription
	b.mtx.Lock()
	for _, sub := range b.subscriptions {
		subs = append(subs, sub)
	}
	b.subscriptions = make(map[string]*Subscription)
	b.mtx.Unlock()

	for _, sub := range subs {
		sub.Close()
	}
}

type eventDispatch struct {
	log    *slog.Logger
	mtx    sync.Mutex
	rt     *Runtime
	topics map[string]*topic
}

func newEvt(logger *slog.Logger, rt *Runtime) *eventDispatch {
	if logger == nil {
		logger = slog.Default()
	}

	return &eventDispatch{
		log:    logger.WithGroup("evt"),
		rt:     rt,
		topics: make(map[string]*topic),
	}
}

func (h *eventDispatch) getTopic(name string) (*topic, bool) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	t, ok := h.topics[name]
	return t, ok
}

func (h *eventDispatch) CreateTopic(name string, bufferSize int) error {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if _, ok := h.topics[name]; ok {
		return fmt.Errorf("topic '%s' already exists", name)
	}

	h.topics[name] = newTopic(bufferSize, h.rt.tracer)
	h.log.Info("topic created", "name", name)
	return nil
}

func (h *eventDispatch) Subscribe(topicName, subscription string) (*Subscription, error) {
	topic, ok := h.getTopic(topicName)
	if !ok {
		return nil, fmt.Errorf("topic '%s' not found", topicName)
	}

	sub, err := topic.Subscribe(subscription)
	if err != nil {
		return nil, err
	}

	h.log.Info("subscribed", "topic", topicName, "subscription", subscription)

	return sub, nil
}

func (h *eventDispatch) Publish(ctx context.Context, topicName string, event any) error {
	topic, ok := h.getTopic(topicName)
	if !ok {
		return fmt.Errorf("topic '%s' not found", topicName)
	}

	err := topic.Publish(ctx, event)

	if err != nil {
		return err
	}

	h.log.Info("published", "topic", topicName, "event", event)

	return nil
}

func (h *eventDispatch) Close() {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	for n, topic := range h.topics {
		topic.Close()
		delete(h.topics, n)
	}
}
