// 🌩 Go++ 🌩 #12 · Concurrency: Fan-out, fan-in
package main

import (
	"fmt"
	"sync"
	"time"
)

// Worker: reads the shared input, sends to its own out, closes out when input is closed.
// Not: forgetting close(out). fanIn waits on it forever.
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

// Fan-in: copies every out into one channel, and closes it once all of them are done.
// Not: close(merged) without wg.Wait(). main ends early and the next send panics.
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

	// Producer: sends every item, then closes input.
	go func() {
		for i := 1; i <= 6; i++ {
			input <- i
		}
		close(input)
	}()

	// Consumer: main closes nothing.
	for s := range fanIn(worker(1, input), worker(2, input), worker(3, input)) {
		fmt.Println(s)
	}
}
