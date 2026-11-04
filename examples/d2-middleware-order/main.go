// 🌩 Go++ 🌩 #22 · Web servers: Middleware order
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

var nextID int

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		nextID++
		c.Set("request_id", fmt.Sprintf("req-%d", nextID))
		c.Next()
	}
}

func logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // a panic below skips every line after this
		fmt.Println("  log:", c.GetString("request_id"), c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	}
}

func newRouter(mw ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(mw...) // the first one wraps everything after it
	r.GET("/panic", func(c *gin.Context) { panic("boom") })
	return r
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	recovery := gin.RecoveryWithWriter(io.Discard) // hide the stack trace in this demo

	// Request ID, then logger, then recovery: the 500 is logged, with an ID.
	good := newRouter(requestID(), logger(), recovery)
	// Not: recovery before the logger, the panic skips the log line.
	bad := newRouter(requestID(), recovery, logger())

	for _, r := range []struct {
		name   string
		router *gin.Engine
	}{{"request ID -> logger -> recovery", good}, {"request ID -> recovery -> logger", bad}} {
		w := httptest.NewRecorder()
		r.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
		fmt.Println(r.name+": client got", w.Code)
	}
}
