// 🌩 Go++ 🌩 #23 · Memory: sync.Pool :: WIP
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
)

// BufferPool reuses *bytes.Buffer and drops any buffer that grew past maxCap,
// so one huge payload can't stay in the pool forever.
type BufferPool struct {
	pool   sync.Pool
	maxCap int
}

func NewBufferPool(maxCap int) *BufferPool {
	p := &BufferPool{maxCap: maxCap}
	p.pool.New = func() any { return new(bytes.Buffer) }
	return p
}

func (p *BufferPool) Get() *bytes.Buffer {
	b := p.pool.Get().(*bytes.Buffer)
	b.Reset() // assume nothing: reset on the way in
	return b
}

func (p *BufferPool) Put(b *bytes.Buffer) {
	if b.Cap() > p.maxCap {
		return // too big to keep, let the GC have it
	}
	p.pool.Put(b)
}

var buffers = NewBufferPool(64 << 10) // keep buffers up to 64 KB

func writeJSON(w http.ResponseWriter, status int, v any) {
	buf := buffers.Get()
	defer buffers.Put(buf) // right after Get, so no path forgets it

	if err := json.NewEncoder(buf).Encode(v); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(status)
	w.Write(buf.Bytes()) // the bytes are copied out before Put runs
}

func main() {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]int{"user_id": 91447})
	fmt.Print(rec.Code, " ", rec.Body.String())
}
