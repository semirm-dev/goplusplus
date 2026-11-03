// 🌩 Go++ 🌩 #19 · Memory: Struct layout
package main

import (
	"fmt"
	"unsafe"
)

// No pointers: the GC skips the whole slice without looking inside.
// Not: Name string; Children []*Node, the GC would follow every pointer.
type Node struct {
	NameOff, NameLen uint32 // where the name sits in Tree.names
	FirstChild, Next int32  // indices into Tree.nodes, -1 means none
}

type Tree struct {
	names []byte // every name, back to back
	nodes []Node
}

func (t *Tree) Add(name string, parent int32) int32 {
	n := Node{NameOff: uint32(len(t.names)), NameLen: uint32(len(name)), FirstChild: -1, Next: -1}
	t.names = append(t.names, name...)
	t.nodes = append(t.nodes, n)
	i := int32(len(t.nodes) - 1)
	if parent >= 0 {
		t.nodes[i].Next = t.nodes[parent].FirstChild
		t.nodes[parent].FirstChild = i
	}
	return i
}

func (t *Tree) Name(i int32) string {
	n := t.nodes[i]
	return string(t.names[n.NameOff : n.NameOff+n.NameLen])
}

// 24 bytes: each bool gets 7 bytes of padding so b can start at 8.
type BadOrder struct {
	a bool  // 1 byte + 7 padding
	b int64 // 8 bytes
	c bool  // 1 byte + 7 padding
}

// 16 bytes: largest field first, the small ones share the rest.
type GoodOrder struct {
	b int64 // 8 bytes
	a bool  // 1 byte
	c bool  // 1 byte + 6 padding
}

func main() {
	var t Tree
	root := t.Add("root", -1)
	t.Add("left", root)
	t.Add("right", root)

	for c := t.nodes[root].FirstChild; c != -1; c = t.nodes[c].Next {
		fmt.Println(t.Name(root), "->", t.Name(c))
	}

	fmt.Println("BadOrder: ", unsafe.Sizeof(BadOrder{}), "bytes")
	fmt.Println("GoodOrder:", unsafe.Sizeof(GoodOrder{}), "bytes")
}
