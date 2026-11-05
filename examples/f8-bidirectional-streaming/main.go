// 🌩 Go++ 🌩 #40 · gRPC: Bidirectional streaming
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	orderv1 "github.com/semirm-dev/goplusplus/examples/f1-code-generation-with-buf/gen/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// server.go

type orderServer struct {
	orderv1.UnimplementedOrderServiceServer
}

func (orderServer) Chat(stream grpc.BidiStreamingServer[orderv1.ChatRequest, orderv1.ChatResponse]) error {
	for {
		// receive a message from the client
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// send a response to the client
			return stream.Send(&orderv1.ChatResponse{Text: "bye"})
		}
		if err != nil {
			return err
		}

		// send a response to the client
		if err := stream.Send(&orderv1.ChatResponse{Text: strings.ToUpper(req.GetText())}); err != nil {
			return err
		}
	}
}

// end

// client.go

func chat(client orderv1.OrderServiceClient, messages []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.Chat(ctx)
	if err != nil {
		return err
	}

	// send messages to the server
	// noticed again? producer started first in goroutine
	go func() {
		for _, text := range messages {
			// send a message to the server
			if err := stream.Send(&orderv1.ChatRequest{Text: text}); err != nil {
				return
			}
		}
		stream.CloseSend()
	}()

	// receive responses from the server
	// consumer started after producer
	for {
		// receive a response from the server
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println(resp.GetText())
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

	if err := chat(orderv1.NewOrderServiceClient(conn), []string{"hello", "how are you", "see you"}); err != nil {
		log.Fatal(err)
	}
}

// end
