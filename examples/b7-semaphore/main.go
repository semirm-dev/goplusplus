// 🌩 Go++ 🌩 #14 · Concurrency: Semaphore
package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Alternative implementation using a struct.
// type SemaphoreLimiter struct {
// 	sem chan struct{}
// }
//
// func NewSemaphoreLimiter(maxConcurrent int) *SemaphoreLimiter {
// 	return &SemaphoreLimiter{
// 		sem: make(chan struct{}, maxConcurrent),
// 	}
// }

// Its capacity is the limit.
type Limiter chan struct{}

func (l Limiter) Acquire(ctx context.Context) error {
	select {
	case l <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l Limiter) Release() {
	<-l
}

func main() {
	limiter := make(Limiter, 3)
	var wg sync.WaitGroup

	for i := 1; i <= 9; i++ {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()

			// If you have thousands of tasks to potentially pile up, call Acquire before the goroutine starts - before wg.Go(func() {}).
			if err := limiter.Acquire(ctx); err != nil {
				fmt.Println("task", i, "gave up:", err)
				return
			}

			defer limiter.Release()

			fmt.Println("task", i, "running")
			time.Sleep(200 * time.Millisecond)
		})
	}

	wg.Wait()
}
