package arise

import (
	"github.com/DarkIntaqt/arise/internal/queues"
	"github.com/DarkIntaqt/arise/internal/task"
)

// NewChannelQueue is a helper function that creates a new instance of ChannelQueue with the specified buffer size.
var NewChannelQueue = queues.NewChannelQueue

type TaskHandler = task.TaskHandler
type TaskQueue = task.TaskQueue
