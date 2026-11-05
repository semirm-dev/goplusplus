// 🌩 Go++ 🌩 #39 · gRPC: Client streaming
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// server.go
// server is listening/receiving from the client

type orderServer struct {
	orderv1.UnimplementedOrderServiceServer
}

func (orderServer) UploadItems(stream grpc.ClientStreamingServer[orderv1.UploadItemsRequest, orderv1.UploadItemsResponse]) error {
	var count int32

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) { // the client is done sending
			return stream.SendAndClose(&orderv1.UploadItemsResponse{Count: count})
		}
		if err != nil {
			return err
		}

		if req.GetQuantity() <= 0 {
			return status.Errorf(codes.InvalidArgument, "item %q: quantity must be > 0", req.GetItem())
		}

		count++
	}
}

// end

// client.go

func uploadItems(client orderv1.OrderServiceClient, items []*orderv1.UploadItemsRequest) (int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.UploadItems(ctx)
	if err != nil {
		return 0, err
	}

	for _, item := range items {
		if err := stream.Send(item); errors.Is(err, io.EOF) {
			break
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return 0, err
	}

	return resp.GetCount(), nil
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

	client := orderv1.NewOrderServiceClient(conn)

	uploads := [][]*orderv1.UploadItemsRequest{
		{{Item: "pen", Quantity: 2}, {Item: "book", Quantity: 1}, {Item: "lamp", Quantity: 3}},
		{{Item: "pen", Quantity: 2}, {Item: "book", Quantity: 0}, {Item: "lamp", Quantity: 3}},
	}

	for _, items := range uploads {
		count, err := uploadItems(client, items)
		if err != nil {
			fmt.Println("upload failed:", status.Code(err), status.Convert(err).Message())
			continue
		}
		fmt.Println("uploaded", count, "items")
	}
}

// end
