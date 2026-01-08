package pubsub

type EventCreateTopic struct {
	Name       string `json:"name"`
	BufferSize uint32 `json:"buffer_size"`
}

func createTopicEvent(name string, bufferSize uint32) EventCreateTopic {
	return EventCreateTopic{
		Name:       name,
		BufferSize: bufferSize,
	}
}

type EventCreateSubscription struct {
	Topic        string `json:"topic"`
	Subscription string `json:"subscription"`
}

func createSubscriptionEvent(topic, subscription string) EventCreateSubscription {
	return EventCreateSubscription{
		Topic:        topic,
		Subscription: subscription,
	}
}

type EventTopicClose struct {
	Topic string `json:"topic"`
}

func topicCloseEvent(topic string) EventTopicClose {
	return EventTopicClose{
		Topic: topic,
	}
}

type EventPublish struct {
	Topic string `json:"topic"`
	Event any    `json:"event"`
}

func publishEvent(topic string, event any) EventPublish {
	return EventPublish{
		Topic: topic,
		Event: event,
	}
}
