// 🌩 Go++ 🌩 #22 · Web servers: Middleware
package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime/debug"
	"strings"
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

func auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") != "Bearer valid-token" { // in production: validate the JWT here
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
		requestID(),                        // first, so everything after it can use the ID
		otelgin.Middleware("orders"),       // tracing
		logger(),                           // logs every request
		recovery(),                         // catch panics in everything after this
		corsPolicy(),                       // apply CORS to your needs
		gzip.Gzip(gzip.DefaultCompression), // compresses responses if the client accepts gzip
		rateLimit(10, 20),                  // add rate limiter to your needs,10 requests per second per IP, bursts up to 20
		maxBody(1024*1024),                 // 1 MB
	)

	r.GET("/panic", func(c *gin.Context) { panic("boom") })

	api := r.Group("/api")
	api.Use(auth())

	api.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetString("user_id")})
	})
	api.POST("/orders", func(c *gin.Context) {
		var order map[string]any
		if err := c.ShouldBindJSON(&order); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "body too large"})
				return
			}
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		c.Status(http.StatusCreated)
	})

	send := func(name, method, path, body string, headers ...string) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		for i := 0; i < len(headers); i += 2 {
			req.Header.Set(headers[i], headers[i+1])
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		fmt.Printf("%-14s -> %d %s\n", name, w.Code, w.Header().Get("Content-Encoding"))
	}

	send("panic", "GET", "/panic", "")
	send("no token", "GET", "/api/me", "")
	send("valid token", "GET", "/api/me", "", "Authorization", "Bearer valid-token")
	send("gzip", "GET", "/api/me", "", "Authorization", "Bearer valid-token", "Accept-Encoding", "gzip")
	send("preflight", "OPTIONS", "/api/me", "", "Origin", "https://app.example.com", "Access-Control-Request-Method", "GET")
	send("2 MB body", "POST", "/api/orders", `{"note":"`+strings.Repeat("x", 2*1024*1024)+`"}`, "Authorization", "Bearer valid-token")

	limited := 0
	for range 20 {
		req := httptest.NewRequest("GET", "/api/me", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			limited++
		}
	}
	fmt.Printf("%-14s -> %d of 20 got 429\n", "flood", limited)
}
