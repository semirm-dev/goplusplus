// 🌩 Go++ 🌩 #20 · Memory: sync.Pool
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
)

var (
	bufferPool = NewBufferPool(0, 64<<10) // 64 KB retain limit
	bytePool   = NewBytePool(32 << 10)    // 32 KB copy buffers
)

// #### Example 1: BufferPool ####

// BufferPool pools *bytes.Buffer values and drops any buffer that grew past maxCap,
// so one huge payload can't pin memory.
type BufferPool struct {
	pool   sync.Pool
	maxCap int
}

// NewBufferPool: initCap pre-sizes new buffers (0 is fine);
// maxCap is the largest capacity kept for reuse.
func NewBufferPool(initCap, maxCap int) *BufferPool {
	p := &BufferPool{maxCap: maxCap}
	p.pool.New = func() any {
		return bytes.NewBuffer(make([]byte, 0, initCap))
	}
	return p
}

func (p *BufferPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

func (p *BufferPool) Put(b *bytes.Buffer) {
	if b.Cap() > p.maxCap {
		return // too big to keep; drop it and let GC free it
	}
	b.Reset() // reset length to 0, keep capacity
	p.pool.Put(b)
}

// BufferPool example: the whole payload in memory, the buffer grows to fit it.
func writeJSON(w http.ResponseWriter, status int, v any) {
	buf := bufferPool.Get()
	defer bufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(v); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(status)
	w.Write(buf.Bytes()) // safe: io.Writer must not retain the slice
}

// #### Example 2: BytePool ####

// BytePool pools fixed-size byte slices. It stores *[]byte because
// putting a plain []byte into any allocates (staticcheck SA6002).
type BytePool struct {
	pool sync.Pool
	size int
}

// NewBytePool: size is the length of every buffer.
func NewBytePool(size int) *BytePool {
	p := &BytePool{size: size}
	p.pool.New = func() any {
		b := make([]byte, size)
		return &b // need to use/return pointer
	}
	return p
}

func (p *BytePool) Get() *[]byte {
	return p.pool.Get().(*[]byte) // remember to use *[]byte
}

func (p *BytePool) Put(b *[]byte) {
	if cap(*b) != p.size {
		return // foreign or re-allocated slice
	}
	*b = (*b)[:p.size] // restore full length (p.size), keep capacity
	p.pool.Put(b)
}

// BytePool example: the payload streams in chunks through a fixed-size buffer.
func copyPooled(dst io.Writer, src io.Reader) (int64, error) {
	bp := bytePool.Get()
	defer bytePool.Put(bp)
	return io.CopyBuffer(dst, src, *bp)
}

func main() {
	// BufferPool
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]int{"user_id": 91447})
	fmt.Print(rec.Code, " ", rec.Body.String())

	// BytePool
	h := sha256.New()
	n, _ := copyPooled(h, io.LimitReader(strings.NewReader("streamed through a pooled buffer"), 1<<20))
	fmt.Printf("%d bytes, sha256 %x\n", n, h.Sum(nil)[:8])
}
