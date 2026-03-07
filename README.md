# Arise

Simple go worker/manager library

## Usage

```go
package main

import (
	"context"
	"time"

	"github.com/DarkIntaqt/arise"
)

type MyTask struct {
	Message string
}

func main() {
	// Create a channel-based queue with a capacity of 50
	queue := arise.NewChannelQueue(50)
	// Pass the queue to the producer
	producer := arise.NewProducer(queue)

	// Create a worker pool from the producer with 5 workers
	pool := arise.FromProducer(producer, arise.PoolOpt{
		Worker: 5,
	})
    
	// Register a handler function for tasks of type MyTask
	arise.RegisterHandler(pool, func(ctx context.Context, task MyTask) error {
		println("Received task with message:", task.Message)
		return nil
	})

	// Start the worker pool and enqueue a task
	pool.Start(context.Background())
	producer.Enqueue(context.Background(), MyTask{Message: "Hello from Arise!"})

	time.Sleep(5 * time.Second)
	// Shutdown the worker pool gracefully
	pool.Shutdown(context.Background())
}

```