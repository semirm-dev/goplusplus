// 🌩 Go++ 🌩 #24 · Web servers: Test handlers without a server
package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Order struct {
	Item     string
	Quantity int
}

var ErrConflict = errors.New("already exists")

type OrderRepository interface {
	Save(ctx context.Context, o Order) error
}

type OrderService struct {
	repo OrderRepository
}

func (s *OrderService) Create(ctx context.Context, o Order) error {
	return s.repo.Save(ctx, o)
}

type createOrderRequest struct {
	Item     string `json:"item" binding:"required"`
	Quantity int    `json:"quantity" binding:"gt=0"`
}

type OrderHandler struct {
	svc *OrderService
}

func (h *OrderHandler) Create(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	order := Order{Item: req.Item, Quantity: req.Quantity}
	err := h.svc.Create(c.Request.Context(), order)

	if errors.Is(err, ErrConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": "already exists"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.Status(http.StatusCreated)
}

func newRouter(svc *OrderService) *gin.Engine {
	r := gin.New()

	h := &OrderHandler{svc: svc}
	r.POST("/orders", h.Create)

	return r
}

func main() {
	// Configure server with newRouter(...) and listen for requests...
}
