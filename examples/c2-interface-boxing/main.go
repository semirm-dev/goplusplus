// 🌩 Go++ 🌩 #17 · Memory: Avoid interface boxing
package main

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
)

var (
	ids     = []int{7, 9999} // read at runtime, so the compiler can't fold them
	sinkAny any
)

func main() {
	small, big := ids[0], ids[1]
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler) // every level is off

	// Not: Info takes ...any, so big is boxed even with the level off.
	fmt.Println("Info:          ", testing.AllocsPerRun(100, func() {
		logger.Info("done", "user_id", big)
	}))
	// Not: slog.Int alone, Info still puts the attr into ...any.
	fmt.Println("Info + Int:    ", testing.AllocsPerRun(100, func() {
		logger.Info("done", slog.Int("user_id", big))
	}))
	fmt.Println("LogAttrs + Int:", testing.AllocsPerRun(100, func() {
		logger.LogAttrs(ctx, slog.LevelInfo, "done", slog.Int("user_id", big))
	}))

	// Testing it: constants point at read-only data and 0-255 come from a static table,
	// so boxing them doesn't allocate. Use a variable above 255.
	fmt.Println("any, const:    ", testing.AllocsPerRun(100, func() { sinkAny = 9999 }))
	fmt.Println("any, 7:        ", testing.AllocsPerRun(100, func() { sinkAny = small }))
	fmt.Println("any, 9999:     ", testing.AllocsPerRun(100, func() { sinkAny = big }))
}
