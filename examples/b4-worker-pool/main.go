// 🌩 Go++ 🌩 #11 · Concurrency: Worker pool
package main

import (
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
func worker(jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	// consumer of jobs
	for job := range jobs {
		// Simulate work
		time.Sleep(50 * time.Millisecond)

		// producer of results
		results <- Result{
			Job:    job,
			Output: job.Input * 2,
		}
	}
}

func main() {
	const numJobs = 10
	const numWorkers = 3

	jobs := make(chan Job, numJobs)
	results := make(chan Result, numJobs)
	var wg sync.WaitGroup

	// 1. Spawn worker pool
	for range numWorkers {
		wg.Add(1)
		go worker(jobs, results, &wg)
	}

	// 2. Stream jobs in the background (PRODUCER of jobs)
	go func() {
		for j := 1; j <= numJobs; j++ {
			jobs <- Job{ID: j, Input: j}
		}
		close(jobs) // Signal workers: no more jobs.
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
