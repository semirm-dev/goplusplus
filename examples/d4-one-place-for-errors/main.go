// 🌩 Go++ 🌩 #24 · Web servers: One place for errors
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

func findUser(id string) (string, error) {
	switch id {
	case "1":
		return "ana", nil
	case "2":
		return "", fmt.Errorf("find user %s: %w", id, ErrNotFound)
	default:
		return "", errors.New(`pq: duplicate key value violates unique constraint "users_email_key"`)
	}
}

// The only place that turns an error into a status code.
func respondError(c *gin.Context, err error) {
	status, msg := http.StatusInternalServerError, "internal error"
	switch {
	case errors.Is(err, ErrNotFound):
		status, msg = http.StatusNotFound, "not found"
	case errors.Is(err, ErrConflict):
		status, msg = http.StatusConflict, "already exists"
	}
	log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err) // the cause goes to the log
	c.JSON(status, gin.H{"error": msg})                                // a fixed message goes to the client
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	log.SetFlags(0)
	r := gin.New()
	r.GET("/users/:id", func(c *gin.Context) {
		name, err := findUser(c.Param("id"))
		if err != nil {
			// Not: c.JSON(500, gin.H{"error": err.Error()}), sends the driver error to the client.
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": name})
	})

	for _, id := range []string{"1", "2", "3"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/"+id, nil))
		fmt.Println("client:", w.Code, w.Body.String())
	}
}
