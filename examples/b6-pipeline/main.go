// 🌩 Go++ 🌩 #13 · Concurrency: Pipeline
package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type Order struct {
	ID     string
	Amount float64
	Total  float64
}

// Stage 1: read raw lines, e.g. a CSV file line by line with bufio.Scanner.
func readLines(ctx context.Context, lines ...string) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		for _, line := range lines {
			select {
			case <-ctx.Done():
				return
			case out <- line:
			}
		}
	}()

	return out
}

// Stage 2: parse each line into an Order, skipping lines that don't parse.
func parse(ctx context.Context, in <-chan string) <-chan Order {
	out := make(chan Order)

	go func() {
		defer close(out)

		for line := range in {
			id, raw, _ := strings.Cut(line, ",")
			amount, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				continue // bad line, skip it
			}

			select {
			case <-ctx.Done():
				return
			case out <- Order{ID: id, Amount: amount}:
			}
		}
	}()

	return out
}

// Stage 3: add 20% tax to each order.
func addTax(ctx context.Context, in <-chan Order) <-chan Order {
	out := make(chan Order)

	go func() {
		defer close(out)

		for o := range in {
			o.Total = o.Amount * 1.2

			select {
			case <-ctx.Done():
				return
			case out <- o:
			}
		}
	}()

	return out
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Assemble the pipeline
	input := readLines(ctx, "A1,10.00", "A2,oops", "A3,25.50", "A4,99.90", "A5,5.00")
	orders := parse(ctx, input)
	results := addTax(ctx, orders)

	for o := range results {
		fmt.Printf("%s: %.2f\n", o.ID, o.Total)
		if o.Total > 100 {
			// Optionally, stop the pipeline early: cancel tells every stage upstream to return.
			cancel()
			break
		}
	}
}
