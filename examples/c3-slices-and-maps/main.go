// 🌩 Go++ 🌩 #18 · Memory: Slices and maps
package main

import "fmt"

type User struct {
	Name string
}

func main() {
	// Slices: allocate the size up front.
	buf := make([]byte, 0, 1024)

	// Reset the length, but keep the capacity, in a for loop.
	for _, chunk := range []string{"first", "second", "third"} {
		buf = buf[:0]
		buf = append(buf, chunk...)
		fmt.Println(string(buf), len(buf), cap(buf)) // cap stays 1024
	}

	// Slice of pointers: set the pointers to nil first, then reset the length.
	users := []*User{{"ana"}, {"ben"}}
	clear(users)
	users = users[:0]
	fmt.Println(len(users), users[:2]) // 0 [<nil> <nil>]

	// Maps: allocate the size up front.
	items := []string{"a", "b", "c"}
	m := make(map[string]int, len(items))
	for i, item := range items {
		m[item] = i
	}

	// Reset the map by re-creating it, clearing it does not free memory.
	m = make(map[string]int, len(items))
	fmt.Println(len(m))
}
