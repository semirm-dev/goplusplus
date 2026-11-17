package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name  string
		path  string
		ready bool
		want  int
	}{
		{"healthz", "/healthz", false, http.StatusOK},
		{"ready", "/readyz", true, http.StatusOK},
		{"shutting down", "/readyz", false, http.StatusServiceUnavailable},
		{"hello", "/v1/hello", true, http.StatusOK},
		{"metrics not on the public port", "/metrics", true, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ready atomic.Bool
			ready.Store(tt.ready)
			w := httptest.NewRecorder()

			newRouter(&ready).ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestMetricsUseRouteTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var ready atomic.Bool
	r := newRouter(&ready)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/hello", nil))

	w := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	want := `http_requests_total{code="200",method="GET",route="/v1/hello"}`
	if !strings.Contains(w.Body.String(), want) {
		t.Fatalf("metrics output has no %s", want)
	}
}

func TestTrustedProxies(t *testing.T) {
	if got := trustedProxies(""); got != nil {
		t.Fatalf("empty = %v, want nil", got)
	}
	if got := trustedProxies("35.191.0.0/16,130.211.0.0/22"); len(got) != 2 || got[1] != "130.211.0.0/22" {
		t.Fatalf("got %v", got)
	}
}
