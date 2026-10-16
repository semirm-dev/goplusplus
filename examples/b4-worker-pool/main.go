// 🌩 Go++ 🌩 #11 · Concurrency: Worker pool
package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Job struct {
	ID    int
	Input int
}

type Result struct {
	Job    Job
	Output int
}

// Worker: CONSUMER of jobs, PRODUCER of results.
func worker(ctx context.Context, input <-chan Job, results chan<- Result) {
	// consumer of jobs
	for job := range input {
		// Simulate work, could be anything: processing a JSON file, an image, etc.
		time.Sleep(50 * time.Millisecond)

		select {
		case <-ctx.Done(): // sending to a channel can be a wait operation, topic #B1 point 3.
			return
		case results <- Result{Job: job, Output: job.Input * 2}: // producer of results
		}
	}
}

func main() {
	input := make(chan Job)
	results := make(chan Result)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Spawn worker pool
	for range 3 {
		wg.Go(func() { worker(ctx, input, results) })
	}

	// 2. Stream jobs in the background (PRODUCER of jobs)
	go func() {
		defer close(input) // Signal workers: no more jobs.

		// this could be anything: many JSON files to read and parse, many images to process, etc.
		for j := 1; j <= 10; j++ {
			select {
			case <-ctx.Done(): // sending to a channel can be a wait operation, topic #B1 point 3.
				return
			case input <- Job{ID: j, Input: j}:
			}
		}
	}()

	// 3. Coordinator: wait for workers to finish, then close results once.
	go func() {
		wg.Wait()
		close(results)
	}()

	// 4. Consume results immediately as they arrive (CONSUMER of results)
	// main consumes here, so every goroutine above must be started first, otherwise it deadlocks.
	for res := range results {
		fmt.Printf("Job %d: Output = %d\n", res.Job.ID, res.Output)
	}
}
