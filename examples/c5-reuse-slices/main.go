// 🌩 Go++ 🌩 #20 · Memory: Reuse capacity, zero the tail :: WIP
package main

import "fmt"

type User struct {
	Name   string
	Active bool
}

// Reuse the backing array: no new allocation for the result.
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
	// Bytes: reslice to zero and refill, the memory is reused.
	buf := make([]byte, 0, 1024)
	for _, chunk := range []string{"first", "second"} {
		buf = buf[:0] // len 0, cap 1024
		buf = append(buf, chunk...)
		fmt.Println(string(buf), len(buf), cap(buf))
	}

	// Pointers: filter in place, then check the tail of the backing array.
	users := []*User{{"ana", true}, {"ben", false}, {"cid", true}, {"dan", false}}
	active := FilterInPlace(users, func(u *User) bool { return u.Active })
	fmt.Println(len(active), "active, tail:", users[2], users[3]) // <nil> <nil>

	// Reset a slice of pointers: clear first, then reslice.
	// Reversed, clear sees len 0 and clears nothing.
	clear(active)
	active = active[:0]
	fmt.Println(len(active), cap(active))
}
