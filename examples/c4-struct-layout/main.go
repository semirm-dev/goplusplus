// 🌩 Go++ 🌩 #19 · Memory: Struct layout
package main

import (
	"fmt"
	"unsafe"
)

// Pointers: the GC looks inside every PtrNode and follows Name and Parent.
type PtrNode struct {
	Name   string
	Parent *PtrNode
}

// No pointers: the GC skips a []Node without looking inside.
type Node struct {
	ID     int64
	Parent int32 // index into the same []Node, -1 means none
}

// Field order: 24 bytes, each bool gets 7 bytes of padding.
type BadOrder struct {
	a bool  // 1 byte + 7 padding
	b int64 // 8 bytes
	c bool  // 1 byte + 7 padding
}

// Biggest field first: 16 bytes.
type GoodOrder struct {
	b int64 // 8 bytes
	a bool  // 1 byte
	c bool  // 1 byte + 6 padding
}

func main() {
	nodes := []Node{
		{ID: 1, Parent: -1},
		{ID: 2, Parent: 0},
		{ID: 3, Parent: 0},
	}
	for _, n := range nodes {
		if n.Parent == -1 {
			continue // skip negative index, out of range [-1]
		}
		fmt.Println(n.ID, "-> parent", nodes[n.Parent].ID)
	}

	fmt.Println("BadOrder: ", unsafe.Sizeof(BadOrder{}), "bytes")
	fmt.Println("GoodOrder:", unsafe.Sizeof(GoodOrder{}), "bytes")
}
