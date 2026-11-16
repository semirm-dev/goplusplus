package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Names and labels match workloads/api/base/prometheusrule.yaml.
var (
	requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests by status code, method and route.",
	}, []string{"code", "method", "route"})
	duration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency by method and route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

func metrics(c *gin.Context) {
	start := time.Now()
	c.Next()

	// The route template, not the raw path: /users/:id is one series, not one per id.
	route := c.FullPath()
	if route == "" {
		route = "unmatched"
	}
	requests.WithLabelValues(strconv.Itoa(c.Writer.Status()), c.Request.Method, route).Inc()
	duration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
}

func newRouter(ready *atomic.Bool) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), metrics)

	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/readyz", func(c *gin.Context) {
		if !ready.Load() {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		c.Status(http.StatusOK)
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/v1/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello"})
	})
	return r
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var ready atomic.Bool
	r := newRouter(&ready)
	// Pod CIDRs: the gateway reaches the pod from inside the cluster.
	if err := r.SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	ready.Store(true)
	log.Println("listening on :" + port)

	<-ctx.Done()
	stop()

	// Fail readiness first and keep serving, so the endpoint is removed before the listener closes.
	ready.Store(false)
	time.Sleep(5 * time.Second)

	// Under the pod's terminationGracePeriodSeconds (30s by default), minus the drain above.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("shutdown:", err)
	}
}
