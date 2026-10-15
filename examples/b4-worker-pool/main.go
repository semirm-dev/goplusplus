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
func worker(ctx context.Context, jobs <-chan Job, results chan<- Result) {
	// consumer of jobs
	for job := range jobs {
		// Simulate work
		time.Sleep(50 * time.Millisecond)

		select {
		case <-ctx.Done():
			return
		case results <- Result{Job: job, Output: job.Input * 2}: // producer of results
		}
	}
}

func main() {
	const numJobs = 10
	const numWorkers = 3

	jobs := make(chan Job, numJobs)
	results := make(chan Result, numJobs)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Spawn worker pool
	for range numWorkers {
		wg.Go(func() { worker(ctx, jobs, results) })
	}

	// 2. Stream jobs in the background (PRODUCER of jobs)
	go func() {
		defer close(jobs) // Signal workers: no more jobs.

		for j := 1; j <= numJobs; j++ {
			select {
			case <-ctx.Done():
				return
			case jobs <- Job{ID: j, Input: j}:
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
