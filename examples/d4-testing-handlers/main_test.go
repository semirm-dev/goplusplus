package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeRepo struct{ err error }

func (f fakeRepo) Save(context.Context, Order) error {
	return f.err
}

func TestCreateOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		body    string
		repoErr error
		want    int
	}{
		{"created", `{"item":"book","quantity":2}`, nil, http.StatusCreated},
		{"invalid", `{"item":"book","quantity":0}`, nil, http.StatusBadRequest},
		{"conflict", `{"item":"book","quantity":2}`, ErrConflict, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &OrderService{repo: fakeRepo{err: tt.repoErr}}
			r := newRouter(svc)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(tt.body))

			r.ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
