// 🌩 Go++ 🌩 #38 · gRPC: Server streaming
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	orderv1 "github.com/semirm-dev/goplusplus/examples/f1-code-generation-with-buf/gen/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// server.go

type orderServer struct {
	orderv1.UnimplementedOrderServiceServer
}

func (orderServer) WatchOrders(_ *orderv1.WatchOrdersRequest, stream grpc.ServerStreamingServer[orderv1.WatchOrdersResponse]) error {
	ctx := stream.Context()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for i := int32(1); ; i++ {
		select {
		case <-ctx.Done():
			fmt.Println("server: client left:", ctx.Err())
			return nil
		case <-ticker.C:
		}

		order := &orderv1.Order{Id: fmt.Sprint(i), Item: "pen", Quantity: i}
		if err := stream.Send(&orderv1.WatchOrdersResponse{Order: order}); err != nil {
			return err
		}
	}
}

// end

// client.go
// client is listening/receiving from the server

func watchOrders(client orderv1.OrderServiceClient, n int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // tells the server to stop sending

	stream, err := client.WatchOrders(ctx, &orderv1.WatchOrdersRequest{})
	if err != nil {
		return err
	}

	for range n {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) { // the server ended the stream
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println("client: got order", resp.GetOrder().GetId())
	}

	return nil
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

	orderv1.RegisterOrderServiceServer(srv, orderServer{})

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

	if err := watchOrders(orderv1.NewOrderServiceClient(conn), 3); err != nil {
		log.Fatal(err)
	}
}

// end
