// 🌩 Go++ 🌩 #24 · Memory: Before you reach for unsafe :: WIP
package main

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"
)

var (
	b       = []byte("hello")
	m       = map[string]int{"hello": 1}
	sinkStr string
	sinkInt int
)

// The bytes must never change while the string is in use.
func BytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func main() {
	// The compiler skips the copy in these four: 0 allocations each.
	fmt.Println("m[string(b)]:    ", testing.AllocsPerRun(100, func() { sinkInt = m[string(b)] }))
	fmt.Println("string(b) == x:  ", testing.AllocsPerRun(100, func() { _ = string(b) == "hello" }))
	fmt.Println("switch string(b):", testing.AllocsPerRun(100, func() {
		switch string(b) {
		case "hello":
			sinkInt++
		}
	}))
	fmt.Println("range string(b): ", testing.AllocsPerRun(100, func() {
		for range string(b) {
			sinkInt++
		}
	}))

	// Keeping the string copies the bytes.
	fmt.Println("keep string(b):  ", testing.AllocsPerRun(100, func() { sinkStr = string(b) }))

	// unsafe: no copy, but b is now read-only.
	fmt.Println("unsafe.String:   ", testing.AllocsPerRun(100, func() { sinkStr = BytesToString(b) }))

	// strings.Builder: one allocation for Grow, String() is a view, no copy.
	parts := []string{"go", "plus", "plus"}
	fmt.Println("strings.Builder: ", testing.AllocsPerRun(100, func() {
		var sb strings.Builder
		sb.Grow(16)
		for _, p := range parts {
			sb.WriteString(p)
		}
		sinkStr = sb.String()
	}))
}
