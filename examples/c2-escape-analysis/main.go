// 🌩 Go++ 🌩 #17 · Memory: Ask the compiler :: WIP
package main

import "fmt"

// Run: go build -gcflags="-m" .
// The comments show what the compiler says about each line.

type User struct {
	ID   int
	Name string
}

func newUser() User {
	return User{ID: 1, Name: "ana"} // stays on the stack: the caller gets a copy
}

func newUserPtr() *User {
	return &User{ID: 2, Name: "ben"} // escapes to heap: the address outlives the function
}

func sumFixed() int {
	buf := make([]byte, 64) // does not escape: constant size
	return len(buf)
}

func sumN(n int) int {
	buf := make([]byte, n) // size unknown: stack if n <= 32 bytes, heap otherwise (Go 1.25+)
	return len(buf)
}

func sumBig() int {
	buf := make([]byte, 128<<10) // escapes to heap: over the 64 KB stack limit
	return len(buf)
}

func counter() func() int {
	n := 0 // moved to heap: the returned closure keeps n alive
	return func() int { n++; return n }
}

func main() {
	u := newUser()
	p := newUserPtr() // inlined here, so &User{...} does not escape anymore
	next := counter()

	// ...any: every argument escapes to heap
	fmt.Println(u.ID, p.ID, sumFixed(), sumN(10), sumBig(), next())
}
