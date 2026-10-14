// 🌩 Go++ 🌩 #10 · Concurrency: Channels vs mutexes
package main

import (
	"fmt"
	"sync"
	"time"
)

// Mutex for state: the map is shared between goroutines.
type Cache struct {
	mu   sync.Mutex
	data map[string]string
}

func (c *Cache) Get(key string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.data[key]
}

func (c *Cache) Refresh(key string) {
	v := fetch(key) // slow I/O, outside the lock

	// Not: c.mu.Lock() before fetch. Every Get waits on the network.
	// Same for disk writes and channel sends.
	c.mu.Lock()
	c.data[key] = v
	c.mu.Unlock()
}

func fetch(key string) string {
	time.Sleep(100 * time.Millisecond) // network call
	return key + "-value"
}

// Channel for coordination: hand each email to the sender, then signal it's done.
func sender(emails <-chan string, done chan<- struct{}) {
	for e := range emails {
		fmt.Println("sent", e)
	}
	close(done)
}

func main() {
	// Mutex: two goroutines write to the same map.
	cache := &Cache{data: map[string]string{}}
	var wg sync.WaitGroup
	for _, k := range []string{"timeout", "region"} {
		wg.Go(func() { cache.Refresh(k) })
	}
	wg.Wait()
	fmt.Println(cache.Get("timeout"), cache.Get("region"))

	// Channel: data passed from one goroutine to another.
	emails := make(chan string)
	done := make(chan struct{})
	go sender(emails, done)

	emails <- "welcome@example.com"
	emails <- "invoice@example.com"
	close(emails)
	<-done
}
