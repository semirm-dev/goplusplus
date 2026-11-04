// 🌩 Go++ 🌩 #21 · Web servers: Configuration and graceful shutdown
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

// Gin has no limit on the request body.
func maxBody(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode) // debug mode is the default
	r := gin.New()

	// By default Gin trusts every proxy, so anyone can fake c.ClientIP() with X-Forwarded-For.
	if err := r.SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil { // your load balancer
		log.Fatal(err)
	}
	r.Use(maxBody(1024 * 1024)) // 1 MB

	r.GET("/slow", func(c *gin.Context) {
		time.Sleep(3 * time.Second) // an in-flight request that Shutdown waits for
		c.String(http.StatusOK, "done\n")
	})

	// Configure your server properly, do not use default one.
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
		// No proxy in front of you? Use srv.ListenAndServeTLS(certFile, keyFile).
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	log.Println("listening on :8080, run: curl localhost:8080/slow, then press Ctrl+C")

	<-ctx.Done()
	stop() // optional: a second Ctrl+C or SIGTERM now kills the process right away
	log.Println("shutting down, waiting for in-flight requests")

	// Keep it under the time your platform gives you before SIGKILL.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("shutdown:", err)
	}
	log.Println("stopped")
}
