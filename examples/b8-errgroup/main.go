// 🌩 Go++ 🌩 #15 · Concurrency: errgroup
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

// Stops when ctx is cancelled, instead of finishing work nobody needs.
func book(ctx context.Context, name string) error {
	select {
	case <-time.After(200 * time.Millisecond):
		fmt.Println("booked", name)
		return nil
	case <-ctx.Done():
		fmt.Println("cancelled", name)
		return ctx.Err()
	}
}

func main() {
	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error { return errors.New("flight: sold out") })
	// Not: book(context.Background(), ...). Nothing could cancel it.
	g.Go(func() error { return book(ctx, "hotel") })
	g.Go(func() error { return book(ctx, "car") })

	fmt.Println(g.Wait())
	// Not: using ctx after Wait. It is already cancelled.
}
