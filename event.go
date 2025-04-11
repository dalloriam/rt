package rt

// A EventType is the type of an event that occurs within the runtime.
type EventType string

const (
	// EventProcessSpawned is emitted when a process is spawned.
	EventTypeProcessSpawned EventType = "process_spawned"

	// EventProcessExited is emitted when a process exits.
	EventTypeProcessExited EventType = "process_exited"

	// EventProcessMessage is emitted when a process sends a message.
	EventTypeClose EventType = "close"

	// EventCreateTopic is emitted when a topic is created.
	EventTypeCreateTopic EventType = "create_topic"

	// EventCreateSubscription is emitted when a subscription is created.
	EventTypeCreateSubscription EventType = "create_subscription"

	// EventTopicClose is emitted when a message is published to a topic.
	EventTypeTopicClose EventType = "topic_close"

	// EventTopicPublish is emitted when a message is published to a topic.
	EventTypePublish EventType = "publish"
)

// An Event is an event that occurs within the runtime.
type BaseEvent struct {
	Type  EventType `json:"type"`
	Error error     `json:"error"`
}

type EventProcessSpawned struct {
	BaseEvent
	PID uint64 `json:"pid"`
}

func processSpawnedEvent(pid uint64) EventProcessSpawned {
	return EventProcessSpawned{
		BaseEvent: BaseEvent{Type: EventTypeProcessSpawned},
		PID:       pid,
	}
}

type EventProcessExited struct {
	BaseEvent
	PID uint32 `json:"pid"`
}

func processExitedEvent(pid uint32, err error) EventProcessExited {
	return EventProcessExited{
		BaseEvent: BaseEvent{Type: EventTypeProcessExited, Error: err},
		PID:       pid,
	}
}

type EventClose struct {
	BaseEvent
}

func closeEvent() EventClose {
	return EventClose{
		BaseEvent: BaseEvent{Type: EventTypeClose},
	}
}

type EventCreateTopic struct {
	BaseEvent
	Name       string `json:"name"`
	BufferSize int    `json:"buffer_size"`
}

func createTopicEvent(name string, bufferSize int, err error) EventCreateTopic {
	return EventCreateTopic{
		BaseEvent:  BaseEvent{Type: EventTypeCreateTopic, Error: err},
		Name:       name,
		BufferSize: bufferSize,
	}
}

type EventCreateSubscription struct {
	BaseEvent
	Topic        string `json:"topic"`
	Subscription string `json:"subscription"`
}

func createSubscriptionEvent(topic, subscription string, err error) EventCreateSubscription {
	return EventCreateSubscription{
		BaseEvent:    BaseEvent{Type: EventTypeCreateSubscription, Error: err},
		Topic:        topic,
		Subscription: subscription,
	}
}

type EventTopicClose struct {
	BaseEvent
	Topic string `json:"topic"`
}

func topicCloseEvent(topic string) EventTopicClose {
	return EventTopicClose{
		BaseEvent: BaseEvent{Type: EventTypeTopicClose},
		Topic:     topic,
	}
}

type EventPublish struct {
	BaseEvent
	Topic string `json:"topic"`
	Event any    `json:"event"`
}

func publishEvent(topic string, event any, err error) EventPublish {
	return EventPublish{
		BaseEvent: BaseEvent{Type: EventTypePublish, Error: err},
		Topic:     topic,
		Event:     event,
	}
}
