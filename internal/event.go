package internal

// A EventType is the type of an event that occurs within the runtime.
type EventType string

const (
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

type EventClose struct {
	BaseEvent
}

func closeEvent() EventClose {
	return EventClose{
		BaseEvent: BaseEvent{Type: EventTypeClose},
	}
}
