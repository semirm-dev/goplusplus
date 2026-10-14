// 🌩 Go++ 🌩 #12 · Concurrency: Fan-out, fan-in
package main

import (
	"fmt"
	"sync"
	"time"
)

// Fan-Out worker
func worker(id int, in <-chan int) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		for n := range in {
			time.Sleep(50 * time.Millisecond) // work
			out <- fmt.Sprintf("worker %d: %d", id, n)
		}
	}()

	return out
}

// Fan-In multiplexer: merges multiple channels into one
func fanIn(outs ...<-chan string) <-chan string {
	merged := make(chan string)
	var wg sync.WaitGroup

	for _, out := range outs {
		wg.Go(func() {
			for s := range out {
				merged <- s
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

	// Simulate input stream
	go func() {
		for i := 1; i <= 6; i++ {
			input <- i
		}
		close(input)
	}()

	// Fan-Out across 3 parallel workers
	w1 := worker(1, input)
	w2 := worker(2, input)
	w3 := worker(3, input)

	// Fan-In to aggregate all output channels
	merged := fanIn(w1, w2, w3)

	for s := range merged {
		fmt.Println(s)
	}
}
