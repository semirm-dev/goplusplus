// 🌩 Go++ 🌩 #22 · Web servers: Middleware
package main

import (
	"crypto/rand"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"golang.org/x/time/rate"
)

// Set request ID so we can trace requests across services.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = rand.Text()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// We can use any logger we want, it's good to log request IDs and other useful request info.
func logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		slog.Info("log-request",
			"id", c.GetString("request_id"),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", time.Since(start).String(),
		)
	}
}

// Log stack trace of error (panic) as JSON so we can parse it from our Log dashboard.
// The client gets simple JSON error as any other 500.
func recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, err any) {
		// for server
		slog.Error("panic",
			"id", c.GetString("request_id"),
			"err", err,
			"stack", string(debug.Stack()),
		)
		// for client
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	})
}

// Configure CORS to your needs.
func corsPolicy() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:  []string{"https://app.example.com"}, // never "*" together with credentials
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:  []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders: []string{"X-Request-ID"},
		MaxAge:        12 * time.Hour, // the browser caches the preflight
	})
}

// This is a simple example, in production you can bring your own rate limiter.
func rateLimit(perSecond rate.Limit, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	clients := map[string]*rate.Limiter{}

	go func() {
		for range time.Tick(time.Minute) {
			mu.Lock()
			clear(clients) // forget old clients, so the map can't grow forever
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		mu.Lock()
		limiter, ok := clients[c.ClientIP()]
		if !ok {
			limiter = rate.NewLimiter(perSecond, burst)
			clients[c.ClientIP()] = limiter
		}
		mu.Unlock()

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}

// It's a good practice to set the body request limit.
// Gin has no limit on the request body.
func maxBody(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
	}
}

// Validate the JWT token here.
func auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") != "Bearer valid-token" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set("user_id", "user-1")
		c.Next()
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	r := gin.New()
	if err := r.SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil { // so c.ClientIP() can't be faked
		log.Fatal(err)
	}

	r.Use(
		requestID(),                        // add request id first
		otelgin.Middleware("orders"),       // tracing
		logger(),                           // log every request
		recovery(),                         // catch panics below
		corsPolicy(),                       // apply CORS
		gzip.Gzip(gzip.DefaultCompression), // (optional) gzip responses
		rateLimit(10, 20),                  // rate limiter
		maxBody(1024*1024),                 // max request body
	)

	r.GET("/panic", func(c *gin.Context) { panic("boom") })

	api := r.Group("/api")
	api.Use(auth())

	// Configure server and listen for requests...
}
