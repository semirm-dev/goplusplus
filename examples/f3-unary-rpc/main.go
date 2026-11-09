// 🌩 Go++ 🌩 #30 · gRPC: Unary RPC
package main

import (
	"context"
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

// server.go

type orderServer struct {
	orderv1.UnimplementedOrderServiceServer
	orders map[string]*orderv1.Order
}

func (s *orderServer) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// the client's deadline came with the call
	if d, ok := ctx.Deadline(); ok {
		fmt.Println("  server: deadline in", time.Until(d).Round(time.Millisecond))
	}

	// get from database... pass ctx, so a cancelled call stops the query too
	select {
	case <-time.After(50 * time.Millisecond):
		fmt.Println("simulate work...database call...")
	case <-ctx.Done():
		return nil, status.FromContextError(ctx.Err()).Err()
	}

	o, ok := s.orders[req.GetId()]
	if !ok {
		// Not: errors.New("not found"), the client gets codes.Unknown
		return nil, status.Errorf(codes.NotFound, "order %s not found", req.GetId())
	}

	return &orderv1.GetOrderResponse{Order: o}, nil
}

// end

// client.go

func getOrder(client orderv1.OrderServiceClient, id string, timeout time.Duration) {
	// always set a deadline, without one a call can wait forever
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := client.GetOrder(ctx, &orderv1.GetOrderRequest{Id: id})

	// check the code, not the error message
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

	orderv1.RegisterOrderServiceServer(srv, &orderServer{
		orders: map[string]*orderv1.Order{"1": {Id: "1", Item: "pen", Quantity: 2}},
	})

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
