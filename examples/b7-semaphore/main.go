// 🌩 Go++ 🌩 #14 · Concurrency: Counting semaphore
package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

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

			if err := limiter.Acquire(ctx); err != nil {
				fmt.Println("task", i, "gave up:", err)
				return
			}
			// Release only what you took.
			// Not: defer limiter.Release() above the error check.
			defer limiter.Release()

			fmt.Println("task", i, "running")
			time.Sleep(200 * time.Millisecond)
		})
	}

	wg.Wait()
}
