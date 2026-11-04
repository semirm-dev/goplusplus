// 🌩 Go++ 🌩 #25 · Web servers: Test handlers without a server
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func newRouter() *gin.Engine {
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func main() {
	newRouter().Run(":8080") // run the tests instead: go test ./examples/d5-test-handlers-without-a-server
}
