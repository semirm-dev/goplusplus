// 🌩 Go++ 🌩 #23 · Web servers: Keep Gin in the handler
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
)

// Domain: no gin, no JSON tags.
type Order struct {
	Item     string
	Quantity int
}

type OrderService struct{}

// ctx is c.Request.Context(): cancelled when the client goes away or the server times out.
func (OrderService) Create(ctx context.Context, o Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fmt.Println("  created", o.Quantity, o.Item)
	return nil
}

// Transport: the request type never leaves the handler.
type createOrderRequest struct {
	Item     string `json:"item" binding:"required"`
	Quantity int    `json:"quantity" binding:"gt=0"`
}

func createOrder(svc OrderService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// Not: svc.Create(c, req), passes gin and the DTO into the service.
		order := Order{Item: req.Item, Quantity: req.Quantity}
		if err := svc.Create(c.Request.Context(), order); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Status(http.StatusCreated)
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.POST("/orders", createOrder(OrderService{}))

	for _, body := range []string{
		`{"item":"book","quantity":2}`,
		`{"item":"book","quantity":0}`, // fails gt=0, never reaches the service
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body)))
		fmt.Println(body, "->", w.Code, w.Body.String())
	}
}
