// 🌩 Go++ 🌩 #9 · Concurrency: Closing channels
package main

import (
	"context"
	"fmt"
	"time"
)

// Producer: owns and closes the channel, respects the consumer's context
func produce(ctx context.Context) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out) // PRODUCER closes the channel when done

		for i := 1; i <= 10; i++ {
			select {
			case <-ctx.Done(): // Responds to consumer cancellation
				return
			case out <- i:
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	return out
}

func main() {
	// Consumer creates and manages the context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := produce(ctx)

	// Consumer decides when to stop
	for n := range stream {
		fmt.Println("Received:", n)
		if n == 3 {
			fmt.Println("Consumer had enough, canceling...")
			cancel() // CONSUMER cancels the context
			break
		}
	}

	// Wait for the producer to exit: range ends when it closes the channel
	for range stream {
	}
	fmt.Println("Producer closed the channel")
}
