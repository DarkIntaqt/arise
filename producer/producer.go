package producer

import (
	"context"
	"reflect"

	"github.com/DarkIntaqt/arise/middleware"
	"github.com/google/uuid"
)

// Producer is responsible for enqueuing tasks into the middleware's TaskQueue.
type Producer struct {
	queue middleware.TaskQueue
}

// NewProducer creates a new Producer with the given TaskQueue.
func NewProducer(q middleware.TaskQueue) *Producer {
	return &Producer{
		queue: q,
	}
}

// Enqueue takes a task of any type, wraps it in a middleware.Task with metadata, and enqueues it into the TaskQueue.
func (p *Producer) Enqueue(ctx context.Context, task any) error {
	name := reflect.TypeOf(task).String()
	return p.queue.Enqueue(ctx, middleware.Task{
		TaskName:   name,
		Payload:    task,
		MaxRetries: 3,
		Retries:    0,
		Id:         uuid.NewString(),
	})
}
