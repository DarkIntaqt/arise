package queues

import (
	"context"
	"sync"

	"github.com/DarkIntaqt/arise/middleware"
)

type ChannelQueue struct {
	pending  chan middleware.Task
	inFlight map[string]middleware.Task
	mu       sync.RWMutex
}

func NewChannelQueue(bufferSize int) *ChannelQueue {
	return &ChannelQueue{
		pending:  make(chan middleware.Task, bufferSize),
		inFlight: make(map[string]middleware.Task),
	}
}

func (c *ChannelQueue) Enqueue(ctx context.Context, task middleware.Task) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.pending <- task:
		return nil
	}
}

func (c *ChannelQueue) Consume(ctx context.Context) (<-chan middleware.Task, error) {
	out := make(chan middleware.Task)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case task, ok := <-c.pending:
				if !ok {
					return
				}

				c.mu.Lock()
				c.inFlight[task.Id] = task
				c.mu.Unlock()

				select {
				case out <- task:
				case <-ctx.Done():
					// If the pool is shutting down and can't take the task, Nack it
					c.Nack(context.Background(), task)
					return
				}
			}
		}
	}()

	return out, nil
}

func (c *ChannelQueue) Ack(ctx context.Context, task middleware.Task) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.inFlight, task.Id)
	return nil
}

func (c *ChannelQueue) Nack(ctx context.Context, task middleware.Task) error {
	c.mu.Lock()
	_, exists := c.inFlight[task.Id]
	if exists {
		delete(c.inFlight, task.Id)
	}
	c.mu.Unlock()

	if exists && task.Retries < task.MaxRetries {
		task.Retries++
		return c.Enqueue(ctx, task)
	}

	return nil
}
