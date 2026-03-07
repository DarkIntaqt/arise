package queues

import (
	"github.com/DarkIntaqt/arise/internal/queues"
)

// NewChannelQueue creates a new ChannelQueue with the specified buffer size.
// This queue uses a buffered channel to hold pending tasks and a map to track in-flight tasks.
// It is not persistent across several instances or restarts.
func NewChannelQueue(bufferSize int) *queues.ChannelQueue {
	return queues.NewChannelQueue(bufferSize)
}
