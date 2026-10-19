// 🌩 Go++ 🌩 #21 · Memory: A slice is a window :: WIP
package main

import (
	"bytes"
	"fmt"
	"runtime"
)

var kept []byte

func heapMB() uint64 {
	runtime.GC()
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	return s.HeapAlloc >> 20
}

// Keeps the whole 64 MB alive: the 4 bytes still point into it.
func headerOf(large []byte) []byte { return large[:4] }

// Copies the 4 bytes, so the 64 MB can be collected.
func headerCopy(large []byte) []byte { return bytes.Clone(large[:4]) }

func main() {
	kept = headerOf(make([]byte, 64<<20))
	fmt.Println("sub-slice kept:", heapMB(), "MB")

	kept = headerCopy(make([]byte, 64<<20))
	fmt.Println("clone kept:    ", heapMB(), "MB")

	// Two-index slice: append writes into the parent's array.
	buf := []byte("hello world")
	sub := buf[0:5]
	_ = append(sub, '!')
	fmt.Println(string(buf)) // hello!world

	// Three-index slice: cap == len, so append has to copy.
	buf = []byte("hello world")
	sub = buf[0:5:5]
	_ = append(sub, '!')
	fmt.Println(string(buf)) // hello world
}
