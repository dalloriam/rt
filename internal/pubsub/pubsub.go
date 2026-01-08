package pubsub

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/dalloriam/rt/api"
	privApi "github.com/dalloriam/rt/internal/api"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// A Subscription represents a subscription to a topic.
// Every message is delivered to every subscription exactly once.
// Multiple consumers can read from the same subscription's Data() channel.
type Subscription struct {
	name     string
	topic    *topic
	rx       chan any
	closed   atomic.Bool
	shutdown chan struct{}
	inflight sync.WaitGroup
}

func newSubscription(name string, t *topic) *Subscription {
	return &Subscription{
		name:     name,
		topic:    t,
		rx:       make(chan any),
		shutdown: make(chan struct{}),
	}
}

// Data returns the channel on which the subscription receives data.
// Consumers should range over this channel to receive messages.
// The channel is closed when the subscription or its associated topic are closed.
func (s *Subscription) Data() <-chan any {
	return s.rx
}

// Close closes the subscription for every consumer.
// Safe to call multiple times per subscription
// (although all subscribers will stop receiving messages once called once).
func (s *Subscription) Close() {
	if s.closed.CompareAndSwap(false, true) {
		s.topic.Unsubscribe(s.name) // ensure no new messages are sent
		close(s.shutdown)
		s.inflight.Wait() // wait for inflight messages to be processed
		close(s.rx)       // we know no new messages will be sent, safe to close
	}
}

type topic struct {
	tracer trace.Tracer
	rx     chan any
	wg     sync.WaitGroup

	mtx           sync.Mutex
	subscriptions map[string]*Subscription
}

func newTopic(bufferSize uint32, tracer trace.Tracer) *topic {
	t := &topic{
		tracer:        tracer,
		rx:            make(chan any, bufferSize),
		subscriptions: make(map[string]*Subscription),
	}

	t.wg.Add(1)
	go t.run()

	return t
}

func (b *topic) run() {
	var dispatchWg sync.WaitGroup

	for evt := range b.rx {
		b.mtx.Lock()
		subs := make([]*Subscription, 0, len(b.subscriptions))
		for _, sub := range b.subscriptions {
			sub.inflight.Add(1)
			subs = append(subs, sub)
		}
		b.mtx.Unlock()

		for _, sub := range subs {
			dispatchWg.Add(1)
			go func(s *Subscription) {
				select {
				case <-s.shutdown:
				case s.rx <- evt:
				}
				dispatchWg.Done()
				s.inflight.Done()
			}(sub)
		}
		dispatchWg.Wait()
	}

	b.wg.Done()
}

// Subscribe creates or retrieves a subscription to the topic.
func (b *topic) Subscribe(subName string) (*Subscription, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if sub, ok := b.subscriptions[subName]; ok {
		return sub, nil
	}

	source := newSubscription(subName, b)
	b.subscriptions[subName] = source

	return source, nil
}

// Unsubscribe removes a subscription from the topic.
func (b *topic) Unsubscribe(subName string) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	delete(b.subscriptions, subName)
}

// Publish sends a message to all subscriptions of the topic.
func (b *topic) Publish(ctx context.Context, msg any) error {
	ctx, span := b.tracer.Start(ctx, "Topic.Publish")
	defer span.End()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case b.rx <- msg:
	}

	return nil
}

// Close closes the topic and all its subscriptions.
func (b *topic) Close() {
	b.mtx.Lock()
	close(b.rx)
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

type PubSub struct {
	log    *slog.Logger
	mtx    sync.Mutex
	rt     privApi.Runtime
	topics map[string]*topic
}

func New(logger *slog.Logger, rt privApi.Runtime) *PubSub {
	if logger == nil {
		logger = slog.Default()
	}

	return &PubSub{
		log:    logger.WithGroup("evt"),
		rt:     rt,
		topics: make(map[string]*topic),
	}
}

func (h *PubSub) getTopic(name string) (*topic, bool) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	t, ok := h.topics[name]
	return t, ok
}

// CreateTopic creates a new topic with the given name and buffer size.
// Returns an error if the topic already exists or if the buffer size exceeds
// the maximum allowed size.
func (h *PubSub) CreateTopic(name string, bufferSize uint32) error {
	if bufferSize > h.rt.Options().PubSub.MaxBufferSize {
		return ErrBufferSizeTooLarge
	}

	if name == "" {
		return ErrInvalidTopicName
	}

	h.mtx.Lock()
	defer h.mtx.Unlock()

	if _, ok := h.topics[name]; ok {
		return fmt.Errorf("topic '%s' already exists", name)
	}

	h.topics[name] = newTopic(bufferSize, noop.NewTracerProvider().Tracer("noop"))
	h.rt.PublishEvent(context.Background(), createTopicEvent(name, bufferSize))
	h.log.Info("topic created", "name", name)
	return nil
}

// Subscribe creates or retrieves a subscription to the given topic.
func (h *PubSub) Subscribe(topicName, subscription string) (api.SubscriptionHandle, error) {
	topic, ok := h.getTopic(topicName)
	if !ok {
		return nil, fmt.Errorf("topic '%s' not found", topicName)
	}

	sub, err := topic.Subscribe(subscription)
	if err != nil {
		return nil, err
	}

	h.log.Info("subscribed", "topic", topicName, "subscription", subscription)
	h.rt.PublishEvent(context.Background(), createSubscriptionEvent(topicName, subscription))

	return sub, nil
}

// Publish sends an event to the given topic.
// Returns an error if the topic does not exist or if publishing fails.
func (h *PubSub) Publish(ctx context.Context, topicName string, msg any) error {
	topic, ok := h.getTopic(topicName)
	if !ok {
		return fmt.Errorf("topic '%s' not found", topicName)
	}

	err := topic.Publish(ctx, msg)

	if err != nil {
		return err
	}

	h.log.Info("published", "topic", topicName, "event", msg)
	h.rt.PublishEvent(ctx, publishEvent(topicName, msg))

	return nil
}

// Close closes all topics and their subscriptions.
func (h *PubSub) Close() {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	for n, topic := range h.topics {
		topic.Close()
		delete(h.topics, n)
	}
}
