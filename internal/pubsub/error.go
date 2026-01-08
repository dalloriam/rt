package pubsub

import "errors"

// ErrBufferSizeTooLarge is returned when the buffer size for a topic exceeds
// the maximum allowed size as defined in the runtime options.
var ErrBufferSizeTooLarge = errors.New("buffer size too large")

// ErrInvalidTopicName is returned when a topic name is invalid.
var ErrInvalidTopicName = errors.New("invalid topic name")
