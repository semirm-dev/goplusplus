// 🌩 Go++ 🌩 #17 · Memory: Avoid interface boxing
package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"testing"
)

var (
	ids     = []int{7, 91447} // read at runtime, so the compiler can't fold them
	sinkAny any
	sinkInt int
	sinkStr string
	sinkBuf []byte
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
	small, id := ids[0], ids[1]
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)
	buf := make([]byte, 0, 64) // owned by you, reused on every call

	fmt.Println("any:            ", testing.AllocsPerRun(100, func() { keepAny(id) }))
	fmt.Println("generic:        ", testing.AllocsPerRun(100, func() { sinkInt = double(id) }))
	fmt.Println("generic + fmt:  ", testing.AllocsPerRun(100, func() { sinkInt = doubleAndFormat(id) }))

	fmt.Println("fmt.Sprintf:    ", testing.AllocsPerRun(100, func() {
		sinkStr = fmt.Sprintf("user=%d", id)
	}))
	fmt.Println("strconv.Append: ", testing.AllocsPerRun(100, func() {
		buf = append(buf[:0], "user="...)
		buf = strconv.AppendInt(buf, int64(id), 10)
		sinkBuf = buf
	}))

	// The level is off, and id is still boxed.
	fmt.Println("log ...any:     ", testing.AllocsPerRun(100, func() {
		logger.Info("processed", "user_id", id)
	}))
	// slog.Int alone isn't enough: Info takes ...any, so the attr is boxed.
	fmt.Println("log Info + Int: ", testing.AllocsPerRun(100, func() {
		logger.Info("processed", slog.Int("user_id", id))
	}))
	fmt.Println("log LogAttrs:   ", testing.AllocsPerRun(100, func() {
		logger.LogAttrs(ctx, slog.LevelInfo, "processed", slog.Int("user_id", id))
	}))

	// The trap: 0-255 come from a static table, so boxing them doesn't allocate.
	fmt.Println("any, value 7:   ", testing.AllocsPerRun(100, func() { keepAny(small) }))
}
