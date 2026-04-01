package common

// MsgPubSub defines the interface for message publish/subscribe operations.
// This abstraction allows different messaging backends (Redis, RocketMQ, etc.)
// to be used interchangeably.
type MsgPubSub interface {
	// Publish publishes a message to a channel/topic.
	// The channel parameter represents the destination (e.g., Redis channel or RocketMQ lite topic).
	Publish(channel string, message string) error

	// Subscribe subscribes to a channel/topic and processes messages.
	// The callback function will be invoked for each received message.
	// The stopChan can be used to signal when to stop the subscription.
	Subscribe(channel string, stopChan chan struct{}, callback func(message string)) error

	// Close closes the connection and releases resources.
	Close() error
}
