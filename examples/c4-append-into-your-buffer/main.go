// 🌩 Go++ 🌩 #19 · Memory: Append into a buffer you own :: WIP
package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"testing"
)

var (
	ids     = []int{91447} // read at runtime, so the compiler can't fold it
	sinkStr string
	sinkBuf []byte
)

func main() {
	id := ids[0]
	ctx := context.Background()
	buf := make([]byte, 0, 64) // owned by you, reused on every call

	// fmt boxes every argument into an any and allocates a new string.
	fmt.Println("fmt.Sprintf:    ", testing.AllocsPerRun(100, func() {
		sinkStr = fmt.Sprintf("user=%d", id)
	}))

	// Append* writes into your buffer and returns it.
	fmt.Println("strconv.Append: ", testing.AllocsPerRun(100, func() {
		buf = append(buf[:0], "user="...)
		buf = strconv.AppendInt(buf, int64(id), 10)
		sinkBuf = buf
	}))

	logger := slog.New(slog.DiscardHandler)

	// ...any: id is boxed before the logger even checks the level.
	fmt.Println("log ...any:     ", testing.AllocsPerRun(100, func() {
		logger.Info("processed", "user_id", id)
	}))

	// Typed attribute: the type travels with the value, no boxing.
	fmt.Println("log typed attr: ", testing.AllocsPerRun(100, func() {
		logger.LogAttrs(ctx, slog.LevelInfo, "processed", slog.Int("user_id", id))
	}))
}
