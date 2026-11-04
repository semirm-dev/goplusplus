// 🌩 Go++ 🌩 #21 · Web servers: Timeouts and graceful shutdown
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/slow", func(c *gin.Context) {
		time.Sleep(2 * time.Second) // an in-flight request that Shutdown waits for
		c.String(http.StatusOK, "done\n")
	})

	// Not: r.Run(":8080"), no timeouts and no way to shut down cleanly.
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,  // slow headers
		ReadTimeout:       10 * time.Second, // slow body
		WriteTimeout:      15 * time.Second, // handler + response
		IdleTimeout:       60 * time.Second, // keep-alive between requests
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	log.Println("listening on :8080, run: curl localhost:8080/slow, then press Ctrl+C")

	<-ctx.Done()
	log.Println("shutting down, waiting for in-flight requests")

	// Keep it under the time your platform gives you before SIGKILL.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("shutdown:", err)
	}
	log.Println("stopped")
}
