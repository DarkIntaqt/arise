package arise

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/DarkIntaqt/arise/internal/task"
)

// PoolState represents the current state of the worker pool.
// A pool can be stopped, running or in the process of shutting down.
type PoolState int32

const (
	// StateStopped indicates that the pool is not running and has no active workers.
	StateStopped PoolState = iota
	// StateRunning indicates that the pool is actively processing tasks with its workers.
	StateRunning
	// StateShuttingDown indicates that the pool is in the process of shutting down, allowing workers to finish current tasks but not accepting new ones.
	StateShuttingDown
)

// Pool manages a set of worker goroutines that consume tasks from a TaskQueue and process them using handler functions
// It can be started and gracefully shut down, ensuring that all in-flight tasks are completed before stopping the workers.
type Pool struct {
	queue     TaskQueue
	workers   uint
	handler   map[string]TaskHandler
	handlerMu sync.RWMutex
	wg        sync.WaitGroup

	state      atomic.Int32
	cancelFunc context.CancelFunc
	shutdown   chan struct{}
	mu         sync.Mutex
}

// PoolOpt defines the configuration options for creating a new worker pool.
type PoolOpt struct {
	// Queue: The TaskQueue from which the pool will consume tasks. This is required and cannot be nil.
	Queue TaskQueue
	// Worker: The number of worker goroutines to start. This must be greater than 0.
	Worker uint
}

// FromProducer creates a new worker pool using the TaskQueue from the provided Producer and the specified PoolOpt configuration.
func FromProducer(p *Producer, opt PoolOpt) *Pool {
	return NewPool(PoolOpt{
		Queue:  p.queue,
		Worker: opt.Worker,
	})
}

func NewPool(opt PoolOpt) *Pool {
	if opt.Queue == nil {
		panic("Queue cannot be nil")
	}

	if opt.Worker == 0 {
		panic("Worker must be greater than 0")
	}

	return &Pool{
		queue:   opt.Queue,
		workers: opt.Worker,
		handler: make(map[string]TaskHandler),
	}
}

// RegisterHandler allows you to register a handler function for a specific task type T.
// The handler will be called with the deserialized task payload when a task of type T is consumed from the queue.
func RegisterHandler[T any](p *Pool, handler func(context.Context, T) error) {
	p.handlerMu.Lock()
	defer p.handlerMu.Unlock()

	name := task.GetTaskName[T]()
	p.handler[name] = func(ctx context.Context, val any) error {
		if typedVal, ok := val.(T); ok {
			return handler(ctx, typedVal)
		}
		return fmt.Errorf("type mismatch for %s", name)
	}
}

// Start initializes the worker pool and begins consuming tasks from the queue.
// Throws an error if the pool is already running or in the process of shutting down.
// The pool will continue to run until Shutdown is called, at which point it will stop accepting new tasks and wait for in-flight tasks to complete before fully stopping.
// The provided context can be used to cancel the all worker goroutines immediately (if implemented by the handlers).
func (p *Pool) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	state := PoolState(p.state.Load())
	if state == StateRunning {
		return errors.New("pool already running")
	}
	if state == StateShuttingDown {
		return errors.New("pool is still shutting down")
	}

	p.shutdown = make(chan struct{})
	workerCtx, cancel := context.WithCancel(ctx)
	p.cancelFunc = cancel

	tasks, err := p.queue.Consume(workerCtx)
	if err != nil {
		cancel()
		return err
	}

	p.state.Store(int32(StateRunning))

	for i := 0; i < int(p.workers); i++ {
		p.wg.Add(1)
		go p.worker(workerCtx, i, tasks)
	}

	return nil
}

// Shutdown gracefully stops the worker pool, allowing in-flight tasks to complete while preventing new tasks from being accepted.
// If the context is canceled before all workers have finished, it will forcefully stop all workers and return an error.
// Waits for all worker goroutines to finish before returning.
func (p *Pool) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	if !p.state.CompareAndSwap(int32(StateRunning), int32(StateShuttingDown)) {
		p.mu.Unlock()
		return errors.New("pool is not running")
	}
	close(p.shutdown)
	p.mu.Unlock()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.state.Store(int32(StateStopped))
		return nil
	case <-ctx.Done():
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.cancelFunc != nil {
			p.cancelFunc()
		}

		p.wg.Wait()
		p.state.Store(int32(StateStopped))
		return ctx.Err()
	}
}

// State returns the current state of the worker pool, which can be Stopped, Running, or ShuttingDown.
func (p *Pool) State() PoolState {
	return PoolState(p.state.Load())
}

// worker is the main loop for each worker goroutine. It listens for tasks on the provided channel and processes them using the registered handlers.
// Additionally, it handles graceful shutdown signals and recovers from panics to ensure the worker pool remains resilient and continues processing tasks even if individual workers encounter errors.
func (p *Pool) worker(ctx context.Context, id int, tasks <-chan task.Task) {
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
		case t, ok := <-tasks:
			if !ok {
				return
			}

			p.handlerMu.RLock()
			handler, exists := p.handler[t.TaskName]
			p.handlerMu.RUnlock()
			if !exists {
				_ = p.queue.Ack(ctx, t)
				continue // silent discard
			}

			if err := handler(ctx, t.Payload); err != nil {
				if nackErr := p.queue.Nack(ctx, t); nackErr != nil {
					fmt.Printf("Worker %d: Nack failed: %v\n", id, nackErr)
				}
			} else {
				if ackErr := p.queue.Ack(ctx, t); ackErr != nil {
					fmt.Printf("Worker %d: Ack failed: %v\n", id, ackErr)
				}
			}
		}
	}
}
