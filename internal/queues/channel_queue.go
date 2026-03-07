package queues

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/DarkIntaqt/arise/internal/task"
)

type ChannelQueue struct {
	pending  chan task.Task
	size     atomic.Int64
	inFlight map[string]task.Task
	mu       sync.RWMutex
}

func NewChannelQueue(bufferSize int) *ChannelQueue {
	return &ChannelQueue{
		pending:  make(chan task.Task, bufferSize),
		inFlight: make(map[string]task.Task),
	}
}

func (c *ChannelQueue) Enqueue(ctx context.Context, task task.Task) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.pending <- task:
		c.size.Add(1)
		return nil
	}
}

func (c *ChannelQueue) Consume(ctx context.Context) (<-chan task.Task, error) {
	out := make(chan task.Task)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case t, ok := <-c.pending:
				if !ok {
					return
				}

				c.size.Add(-1)
				c.mu.Lock()
				c.inFlight[t.Id] = t
				c.mu.Unlock()

				select {
				case out <- t:
				case <-ctx.Done():
					// If the pool is shutting down and can't take the task, Nack it
					c.Nack(context.Background(), t)
					return
				}
			}
		}
	}()

	return out, nil
}

func (c *ChannelQueue) Ack(ctx context.Context, task task.Task) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.inFlight, task.Id)
	return nil
}

func (c *ChannelQueue) Nack(ctx context.Context, task task.Task) error {
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

func (c *ChannelQueue) Size() int64 {
	return c.size.Load()
}
