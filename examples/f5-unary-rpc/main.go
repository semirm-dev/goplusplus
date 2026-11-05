// 🌩 Go++ 🌩 #37 · gRPC: Unary RPC
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	orderv1 "github.com/semirm-dev/goplusplus/examples/f1-code-generation-with-buf/gen/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// store.go
// knows nothing about gRPC

var ErrNotFound = errors.New("not found")

type Order struct {
	ID       string
	Item     string
	Quantity int32
}

type Store struct {
	orders map[string]Order
}

func (s *Store) Find(ctx context.Context, id string) (Order, error) {
	select {
	case <-time.After(50 * time.Millisecond): // simulate database call
	case <-ctx.Done():
		return Order{}, ctx.Err()
	}

	o, ok := s.orders[id]
	if !ok {
		return Order{}, fmt.Errorf("order %s: %w", id, ErrNotFound)
	}
	return o, nil
}

// end

// server.go

type orderServer struct {
	orderv1.UnimplementedOrderServiceServer
	store *Store
}

func (s *orderServer) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if d, ok := ctx.Deadline(); ok {
		fmt.Printf("  server: deadline in %v\n", time.Until(d).Round(time.Millisecond))
	}

	o, err := s.store.Find(ctx, req.GetId())
	switch {
	case errors.Is(err, ErrNotFound):
		return nil, status.Errorf(codes.NotFound, "order %s not found", req.GetId())
	case ctx.Err() != nil:
		return nil, status.FromContextError(ctx.Err()).Err()
	case err != nil:
		log.Printf("get order %s: %v", req.GetId(), err)
		return nil, status.Error(codes.Internal, "internal error") // don't leak internals to the client
	}

	return &orderv1.GetOrderResponse{Order: &orderv1.Order{Id: o.ID, Item: o.Item, Quantity: o.Quantity}}, nil
}

// end

// client.go
// use the client to call the server

func getOrder(client orderv1.OrderServiceClient, id string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := client.GetOrder(ctx, &orderv1.GetOrderRequest{Id: id})

	switch status.Code(err) {
	case codes.OK:
		fmt.Printf("%q: %s x%d\n", id, resp.GetOrder().GetItem(), resp.GetOrder().GetQuantity())
	case codes.NotFound:
		fmt.Printf("%q: not found\n", id)
	default:
		fmt.Printf("%q: %v\n", id, status.Code(err))
	}
}

// end

// main.go

func main() {
	lis, err := net.Listen("tcp", "localhost:50051")
	if err != nil {
		log.Fatal(err)
	}

	srv := grpc.NewServer()
	defer srv.GracefulStop()

	store := &Store{orders: map[string]Order{"1": {ID: "1", Item: "pen", Quantity: 2}}}

	orderv1.RegisterOrderServiceServer(srv, &orderServer{store: store})

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	// Local example. In production use TLS: credentials.NewTLS(cfg).
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := orderv1.NewOrderServiceClient(conn)

	getOrder(client, "1", 2*time.Second)
	getOrder(client, "", 2*time.Second)
	getOrder(client, "2", 2*time.Second)
	getOrder(client, "1", 10*time.Millisecond) // shorter than the database call
}

// end
