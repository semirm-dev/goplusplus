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
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stream.Send(&orderv1.ChatResponse{Text: "bye"})
		}
		if err != nil {
			return err
		}

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

	go func() {
		for _, text := range messages {
			if err := stream.Send(&orderv1.ChatRequest{Text: text}); err != nil {
				return
			}
		}
		stream.CloseSend()
	}()

	for {
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
