// 🌩 Go++ 🌩 #16 · Memory: Measure, tune, rewrite
package main

// Run: go build -gcflags="-m" ./escape

var sink int

type User struct {
	ID   int
	Name string
}

func newUser() User {
	return User{ID: 1, Name: "ana"} // stays on the stack: the caller gets a copy
}

func newUserPtr() *User {
	return &User{ID: 2, Name: "ben"} // escapes to heap: the pointer outlives the function
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
	buf := make([]byte, 128*1024) // escapes to heap: over the 64 KB stack limit
	return len(buf)
}

func counter() func() int {
	n := 0 // moved to heap: the returned closure keeps n alive
	return func() int { n++; return n }
}

func main() {
	// newUserPtr is inlined: its body is copied here.
	// The User never leaves main, so it stays on the stack.
	p := newUserPtr()
	sink = p.ID
}
