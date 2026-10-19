// 🌩 Go++ 🌩 #25 · Memory: Maps never shrink :: WIP
package main

import (
	"fmt"
	"runtime"
)

var m map[int64]int64

func heapMB() uint64 {
	runtime.GC()
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	return s.HeapAlloc >> 20
}

func main() {
	const n = 1_000_000

	m = make(map[int64]int64, n) // size hint: no growing on the way
	for i := range int64(n) {
		m[i] = i
	}
	fmt.Println("filled:     ", heapMB(), "MB")

	for k := range m {
		delete(m, k)
	}
	fmt.Println("deleted all:", heapMB(), "MB, len", len(m))

	clear(m) // empties it, keeps the memory
	fmt.Println("clear(m):   ", heapMB(), "MB")

	m = make(map[int64]int64) // a new map is the only way to give it back
	fmt.Println("new map:    ", heapMB(), "MB")
}
