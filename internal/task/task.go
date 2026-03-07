package task

import (
	"context"
	"reflect"
)

// Task represents a unit of work to be processed by the worker pool.
type Task struct {
	// TaskName is the name of the task, used to route it to the appropriate handler.
	// It gets automatically set to the type name of the payload when enqueued via the Producer.
	TaskName string
	// Payload is the actual data for the task.
	//It can be of any type, but it's typically a struct that contains the necessary information for processing.
	Payload any
	// Id is a unique identifier for the task, which can be used for tracking and logging purposes.
	// It gets automatically generated when enqueued via the Producer (uuid)
	Id string
	// Retries keeps track of how many times the task has been retried after a failure.
	Retries uint
	// MaxRetries defines the maximum number of retry attempts allowed for the task before it is considered failed.
	MaxRetries uint
}

// TaskHandler is a function type that defines the signature for task processing functions.
type TaskHandler func(ctx context.Context, payload any) error

// TaskQueue is an interface that abstracts the underlying task queue implementation.
type TaskQueue interface {
	// Enqueue adds a new task to the queue for processing.
	// It takes a context for cancelling the enqueue operation and a Task struct that contains the task details.
	Enqueue(ctx context.Context, task Task) error
	// Consume returns a channel from which tasks can be read for processing.
	// It takes a context for cancelling the consume operation and returns a read-only channel of Tasks and an error if the operation fails.
	Consume(ctx context.Context) (<-chan Task, error)
	// Ack acknowledges that a task has been successfully processed and can be removed from the queue.
	// It takes a context for cancelling the ack operation and the Task that was processed.
	Ack(ctx context.Context, task Task) error
	// Nack indicates that a task has failed to process and should be retried or discarded based on the retry policy.
	// It takes a context for cancelling the nack operation and the Task that failed to process.
	Nack(ctx context.Context, task Task) error
	// Size returns the current number of pending tasks in the queue
	Size() int64
}

func GetTaskName[T any]() string {
	return reflect.TypeOf((*T)(nil)).Elem().String()
}
