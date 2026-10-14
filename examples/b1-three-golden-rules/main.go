// 🌩 Go++ 🌩 #8 · Concurrency: Three golden rules
package main

import (
	"context"
	"fmt"
)

// #2: It returns when ch is closed or ctx is cancelled.
// The loop has an exit plan, and the goroutine will not hang forever.
//
// #3: Context as the first argument.
func consume(ctx context.Context, ch <-chan int) {
	for {
		select {
		case <-ctx.Done():
			return
		case n, ok := <-ch:
			if !ok {
				return
			}
			fmt.Println("consumed", n)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan int)
	done := make(chan struct{})

	// #1: The caller produces, so the consumer starts first.
	// We will not send values before the consumer is started, otherwise it will deadlock.
	go func() {
		consume(ctx, ch)
		close(done)
	}()

	for i := 1; i <= 3; i++ {
		// Safe to send: the consumer already exists and will receive values.
		ch <- i
	}
	close(ch)

	<-done
}
