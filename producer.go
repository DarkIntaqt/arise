package arise

import (
	"context"
	"reflect"

	"github.com/DarkIntaqt/arise/internal/task"
	"github.com/google/uuid"
)

// Producer is responsible for enqueuing tasks into the middleware's TaskQueue.
type Producer struct {
	queue TaskQueue
}

// NewProducer creates a new Producer with the given TaskQueue.
func NewProducer(q TaskQueue) *Producer {
	return &Producer{
		queue: q,
	}
}

// Enqueue takes a task of any type, wraps it in a middleware.Task with metadata, and enqueues it into the TaskQueue.
func (p *Producer) Enqueue(ctx context.Context, t any) error {
	name := reflect.TypeOf(t).String()
	return p.queue.Enqueue(ctx, task.Task{
		TaskName:   name,
		Payload:    t,
		MaxRetries: 3,
		Retries:    0,
		Id:         uuid.NewString(),
	})
}
