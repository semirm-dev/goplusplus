// 🌩 Go++ 🌩 #18 · Memory: any allocates :: WIP
package main

import (
	"fmt"
	"testing"
)

var (
	ids     = []int{7, 91447} // read at runtime, so the compiler can't fold them
	sinkAny any
	sinkInt int
	sinkStr string
)

// Boxes v: an interface holds a pointer, so v is copied to the heap.
func keepAny(v any) { sinkAny = v }

// No boxing: T stays a concrete type.
func double[T int | float64](v T) T { return v * 2 }

// Boxes again: fmt takes ...any.
func doubleAndFormat[T int | float64](v T) T {
	sinkStr = fmt.Sprint(v)
	return v * 2
}

func main() {
	small, large := ids[0], ids[1]

	fmt.Println("any, value 91447:", testing.AllocsPerRun(100, func() { keepAny(large) }))
	fmt.Println("generic:         ", testing.AllocsPerRun(100, func() { sinkInt = double(large) }))
	fmt.Println("generic + fmt:   ", testing.AllocsPerRun(100, func() { sinkInt = doubleAndFormat(large) }))

	// The trap: 0-255 come from a static table, so boxing them doesn't allocate.
	fmt.Println("any, value 7:    ", testing.AllocsPerRun(100, func() { keepAny(small) }))
}
