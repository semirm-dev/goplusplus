// 🌩 Go++ 🌩 #26 · Memory: Struct padding :: WIP
package main

import (
	"fmt"
	"unsafe"
)

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
	var bad BadOrder
	var good GoodOrder

	fmt.Println("BadOrder: ", unsafe.Sizeof(bad), "bytes, b at offset", unsafe.Offsetof(bad.b))
	fmt.Println("GoodOrder:", unsafe.Sizeof(good), "bytes, b at offset", unsafe.Offsetof(good.b))
}
