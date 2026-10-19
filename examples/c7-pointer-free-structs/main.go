// 🌩 Go++ 🌩 #22 · Memory: Pointer-free structs :: WIP
package main

import "fmt"

// The GC follows every pointer here: the string and each child.
type PtrNode struct {
	Name     string
	Children []*PtrNode
}

// No pointers: the GC skips the whole slice without looking inside.
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

func main() {
	var t Tree
	root := t.Add("root", -1)
	t.Add("left", root)
	t.Add("right", root)

	for c := t.nodes[root].FirstChild; c != -1; c = t.nodes[c].Next {
		fmt.Println(t.Name(root), "->", t.Name(c))
	}

	_ = PtrNode{Name: "same tree, but traced by the GC"}
}
