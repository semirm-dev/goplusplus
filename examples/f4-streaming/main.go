// 🌩 Go++ 🌩 #31 · gRPC: Streaming
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	orderv1 "github.com/semirm-dev/goplusplus/examples/f1-code-generation-with-buf/gen/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type orderServer struct {
	orderv1.UnimplementedOrderServiceServer
}

// ## 1. server streaming: one request, many responses

// server.go

func (orderServer) WatchOrders(_ *orderv1.WatchOrdersRequest, stream grpc.ServerStreamingServer[orderv1.WatchOrdersResponse]) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for i := int32(1); ; i++ {
		select {
		case <-stream.Context().Done(): // the client cancelled or left
			return nil
		case <-ticker.C:
		}

		order := &orderv1.Order{Id: fmt.Sprint(i), Item: "pen", Quantity: i}
		// send to the client
		if err := stream.Send(&orderv1.WatchOrdersResponse{Order: order}); err != nil {
			return err
		}
	}
}

// client.go

func watchOrders(client orderv1.OrderServiceClient) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // the consumer cancels, the server (producer) stops

	stream, err := client.WatchOrders(ctx, &orderv1.WatchOrdersRequest{})
	if err != nil {
		return err
	}

	// read from server
	for range 3 {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) { // the server ended the stream
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println("watch: order", resp.GetOrder().GetId())
	}
	return nil
}

// ## end

// ## 2. client streaming: many requests, one response

// server.go

func (orderServer) UploadItems(stream grpc.ClientStreamingServer[orderv1.UploadItemsRequest, orderv1.UploadItemsResponse]) error {
	var count int32
	for {
		// receive from the client
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

// client.go

func uploadItems(client orderv1.OrderServiceClient, items []*orderv1.UploadItemsRequest) (int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.UploadItems(ctx)
	if err != nil {
		return 0, err
	}

	for _, item := range items {
		// io.EOF: the server already ended the stream, the real error comes from CloseAndRecv
		// send to the server
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

// ## end

// ## 3. bidirectional streaming: both sides send whenever they want

// server.go

func (orderServer) Chat(stream grpc.BidiStreamingServer[orderv1.ChatRequest, orderv1.ChatResponse]) error {
	for {
		// receive from the client
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) { // the client is done sending, we can still send
			return stream.Send(&orderv1.ChatResponse{Text: "bye"})
		}
		if err != nil {
			return err
		}

		// send to the client
		if err := stream.Send(&orderv1.ChatResponse{Text: strings.ToUpper(req.GetText())}); err != nil {
			return err
		}
	}
}

// client.go

func chat(client orderv1.OrderServiceClient, messages []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.Chat(ctx)
	if err != nil {
		return err
	}

	// one goroutine sends, another receives
	// noticed again? producer started first in goroutine
	// Not: two goroutines calling Send on the same stream
	go func() {
		for _, text := range messages {
			// send to server
			if err := stream.Send(&orderv1.ChatRequest{Text: text}); err != nil {
				return
			}
		}
		stream.CloseSend() // from the same goroutine that sends
	}()

	// consumer started after producer's goroutine
	for {
		// receive from server
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println("chat:", resp.GetText())
	}
}

// ## end

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

	serverStreaming(client)
	clientStreaming(client)
	bidiStreaming(client)
}

func serverStreaming(client orderv1.OrderServiceClient) {
	if err := watchOrders(client); err != nil {
		log.Fatal(err)
	}
}

func clientStreaming(client orderv1.OrderServiceClient) {
	uploads := [][]*orderv1.UploadItemsRequest{
		{{Item: "pen", Quantity: 2}, {Item: "book", Quantity: 1}},
		{{Item: "pen", Quantity: 2}, {Item: "book", Quantity: 0}},
	}
	for _, items := range uploads {
		count, err := uploadItems(client, items)
		if err != nil {
			fmt.Println("upload: failed,", status.Code(err), status.Convert(err).Message())
			continue
		}
		fmt.Println("upload:", count, "items")
	}
}

func bidiStreaming(client orderv1.OrderServiceClient) {
	if err := chat(client, []string{"hello", "how are you"}); err != nil {
		log.Fatal(err)
	}
}

// end
