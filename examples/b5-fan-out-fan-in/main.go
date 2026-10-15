// 🌩 Go++ 🌩 #12 · Concurrency: Fan-out, fan-in
package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Fan-Out worker: one of several copies reading the same input.
// The job could be anything: parsing many JSON files, processing many images, etc.
func worker(ctx context.Context, id int, in <-chan int) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		for n := range in {
			time.Sleep(50 * time.Millisecond) // work

			select {
			case <-ctx.Done():
				return
			case out <- fmt.Sprintf("worker %d: %d", id, n):
			}
		}
	}()

	return out
}

// Fan-In multiplexer: merges multiple channels into one
func fanIn(ctx context.Context, outs ...<-chan string) <-chan string {
	merged := make(chan string)
	var wg sync.WaitGroup

	for _, out := range outs {
		wg.Go(func() {
			for s := range out {
				select {
				case <-ctx.Done():
					return
				case merged <- s:
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(merged)
	}()

	return merged
}

func main() {
	input := make(chan int)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Simulate input stream
	go func() {
		defer close(input)

		for i := 1; i <= 6; i++ {
			select {
			case <-ctx.Done():
				return
			case input <- i:
			}
		}
	}()

	// Fan-Out across 3 parallel workers
	w1 := worker(ctx, 1, input)
	w2 := worker(ctx, 2, input)
	w3 := worker(ctx, 3, input)

	// Fan-In to aggregate all output channels
	merged := fanIn(ctx, w1, w2, w3)

	for s := range merged {
		fmt.Println(s)
	}
}
