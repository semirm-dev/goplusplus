// 🌩 Go++ 🌩 #13 · Concurrency: Pipeline
package main

import (
	"context"
	"fmt"
)

// Source: sends every value, then closes out.
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

// Stage: the function that returns a channel is the one that closes it.
// Not: a plain out <- n. If main stops reading, this blocks forever.
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

	for n := range square(ctx, source(ctx, 1, 2, 3, 4, 5)) {
		fmt.Println(n)
		if n == 9 {
			// Stopping early: cancel tells every stage upstream to return.
			cancel()
			break
		}
	}
}
