// 🌩 Go++ 🌩 #23 · Web servers: Handlers
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
)

// order.go
// plain Go structs

// Domain: no gin, no JSON tags.
type Order struct {
	Item     string
	Quantity int
}

// end

// service.go
// uses domain
// knows nothing about gin

var ErrConflict = errors.New("already exists")

type OrderService struct{}

// Go's context, not Gin's!!
func (OrderService) Create(ctx context.Context, o Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch o.Item {
	case "pen":
		return fmt.Errorf("create order %s: %w", o.Item, ErrConflict)
	case "lamp":
		return errors.New(`pq: duplicate key value violates unique constraint "orders_item_key"`)
	}
	fmt.Println("  created", o.Quantity, o.Item)
	return nil
}

// end

// handler.go
// uses service

var ErrInvalidInput = errors.New("invalid request")

// Transport: the request type never leaves the handler.
type createOrderRequest struct {
	Item     string `json:"item" binding:"required"`
	Quantity int    `json:"quantity" binding:"gt=0"`
}

type OrderHandler struct {
	svc OrderService
}

func (h *OrderHandler) Create(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		err = fmt.Errorf("%w: %v", ErrInvalidInput, err)
		respondError(c, err)
		return
	}

	// Not: h.svc.Create(c, req), because then it passes gin and the request type into the service.
	order := Order{Item: req.Item, Quantity: req.Quantity}
	err := h.svc.Create(c.Request.Context(), order)

	if err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

// shared and reusable error handling logic
func respondError(c *gin.Context, err error) {
	status, msg := http.StatusInternalServerError, "internal error"
	switch {
	case errors.Is(err, ErrInvalidInput):
		status, msg = http.StatusBadRequest, "invalid request"
	case errors.Is(err, ErrConflict):
		status, msg = http.StatusConflict, "already exists"
	}

	log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err) // the cause goes to the log

	// Not: c.JSON(500, gin.H{"error": err.Error()}), sends the driver error to the client.
	c.JSON(status, gin.H{"error": msg})
}

// end

func main() {
	gin.SetMode(gin.ReleaseMode)
	log.SetFlags(0)
	r := gin.New()

	h := &OrderHandler{svc: OrderService{}}

	r.POST("/orders", h.Create)

	for _, body := range []string{
		`{"item":"book","quantity":2}`,
		`{"item":"book","quantity":0}`, // fails gt=0, never reaches the service
		`{"item":"pen","quantity":1}`,  // ErrConflict -> 409
		`{"item":"lamp","quantity":1}`, // unknown error -> 500, the driver error stays in the log
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body)))
		fmt.Println(body, "->", w.Code, w.Body.String())
	}
}
