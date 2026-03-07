package worker

import (
	"context"
	"fmt"

	"github.com/DarkIntaqt/arise/middleware"
)

// worker is the main loop for each worker goroutine. It listens for tasks on the provided channel and processes them using the registered handlers.
// Additionally, it handles graceful shutdown signals and recovers from panics to ensure the worker pool remains resilient and continues processing tasks even if individual workers encounter errors.
func (p *Pool) worker(ctx context.Context, id int, tasks <-chan middleware.Task) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Worker %d panicked: %v. Restarting...\n", id, r)
			go p.worker(ctx, id, tasks) // Restart the worker
		} else {
			p.wg.Done()
		}
	}()

	for {
		select {
		case <-p.shutdown:
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}

			p.handlerMu.RLock()
			handler, exists := p.handler[task.TaskName]
			p.handlerMu.RUnlock()
			if !exists {
				_ = p.queue.Ack(ctx, task)
				continue // silent discard
			}

			if err := handler(ctx, task.Payload); err != nil {
				if nackErr := p.queue.Nack(ctx, task); nackErr != nil {
					fmt.Printf("Worker %d: Nack failed: %v\n", id, nackErr)
				}
			} else {
				if ackErr := p.queue.Ack(ctx, task); ackErr != nil {
					fmt.Printf("Worker %d: Ack failed: %v\n", id, ackErr)
				}
			}
		}
	}
}
