// 🌩 Go++ 🌩 #13 · Concurrency: Pipeline
package main

import (
	"context"
	"fmt"
)

// Stage 1: produce values, then close its out channel.
func source(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

// Stage 2: transform values, then close its out channel.
func square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for n := range in {
			select {
			case out <- n * n:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Assemble the pipeline
	numbers := source(ctx, 1, 2, 3, 4, 5)
	squared := square(ctx, numbers)

	for n := range squared {
		fmt.Println(n) // 1, 4, 9, 16, 25
		if n == 9 {
			// Stopping early: cancel tells every stage upstream to return.
			cancel()
			break
		}
	}
}
