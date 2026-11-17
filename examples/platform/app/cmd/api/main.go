package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	r.GET("/v1/hello", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello"})
	})
	return r
}

// trustedProxies parses TRUSTED_PROXIES: the hops allowed to set X-Forwarded-For.
// Empty trusts none, so the client IP is the peer address.
func trustedProxies(env string) []string {
	if env == "" {
		return nil
	}
	return strings.Split(env, ",")
}

func newServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func serve(srv *http.Server) {
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("serve", "addr", srv.Addr, "err", err)
		os.Exit(1)
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	gin.SetMode(gin.ReleaseMode)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var ready atomic.Bool
	r := newRouter(&ready)
	if err := r.SetTrustedProxies(trustedProxies(os.Getenv("TRUSTED_PROXIES"))); err != nil {
		slog.Error("trusted proxies", "err", err)
		os.Exit(1)
	}

	srv := newServer(":"+port, r)
	// Metrics on their own port: the HTTPRoute only reaches 8080, so /metrics is never public.
	metricsSrv := newServer(":9090", promhttp.Handler())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go serve(srv)
	go serve(metricsSrv)
	ready.Store(true)
	slog.Info("listening", "addr", srv.Addr, "metrics", metricsSrv.Addr)

	<-ctx.Done()
	stop()

	// Fail readiness first and keep serving, so the endpoint is removed before the listener closes.
	ready.Store(false)
	time.Sleep(5 * time.Second)

	// Under the pod's terminationGracePeriodSeconds (30s by default), minus the drain above.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
	// Last, so the final requests are still scrapeable while the API drains.
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown metrics", "err", err)
	}
}
