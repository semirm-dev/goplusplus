// 🌩 Go++ 🌩 #18 · Memory: Slices and maps
package main

import (
	"bytes"
	"fmt"
	"runtime"
)

var (
	kept []byte
	m    map[int64]int64
)

type User struct {
	Name   string
	Active bool
}

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

// Reuses the backing array: no new allocation for the result.
func FilterInPlace(users []*User, keep func(*User) bool) []*User {
	filtered := users[:0] // same backing array, len 0
	for _, u := range users {
		if keep(u) {
			filtered = append(filtered, u)
		}
	}
	clear(users[len(filtered):]) // drop the pointers the GC would still see
	return filtered
}

func main() {
	kept = headerOf(make([]byte, 64<<20))
	fmt.Println("sub-slice kept:", heapMB(), "MB")

	kept = headerCopy(make([]byte, 64<<20))
	fmt.Println("clone kept:    ", heapMB(), "MB")

	// Bytes: reslice to zero and refill, the memory is reused.
	buf := make([]byte, 0, 1024)
	buf = append(buf[:0], "reused"...)
	fmt.Println(string(buf), len(buf), cap(buf))

	users := []*User{{"ana", true}, {"ben", false}, {"cid", true}, {"dan", false}}
	active := FilterInPlace(users, func(u *User) bool { return u.Active })
	fmt.Println(len(active), "active, tail:", users[2], users[3]) // <nil> <nil>

	// Reset a slice of pointers: clear first, then reslice.
	// Not: active = active[:0]; clear(active), clear sees len 0 and clears nothing.
	clear(active)
	active = active[:0]

	m = make(map[int64]int64)
	for i := range int64(1_000_000) {
		m[i] = i
	}
	fmt.Println("map filled:    ", heapMB(), "MB")

	for k := range m {
		delete(m, k) // clear(m) is the same: empty, but the memory stays
	}
	fmt.Println("map emptied:   ", heapMB(), "MB")

	m = make(map[int64]int64) // a new map is the only way to give it back
	fmt.Println("new map:       ", heapMB(), "MB")
}
