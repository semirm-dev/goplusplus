// 🌩 Go++ 🌩 #29 · gRPC: gRPC server
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	orderv1 "github.com/semirm-dev/goplusplus/examples/f1-code-generation-with-buf/gen/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// server.go

// WatchOrders, UploadItems and Chat return codes.Unimplemented until you write them.
type orderServer struct {
	orderv1.UnimplementedOrderServiceServer
}

func (orderServer) GetOrder(_ context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	return &orderv1.GetOrderResponse{Order: &orderv1.Order{Id: req.GetId(), Item: "pen", Quantity: 2}}, nil
}

// end

// main.go

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	srv := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(srv, orderServer{})

	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)
	reflection.Register(srv)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()
	log.Println("listening on :50051, run: grpcurl -plaintext localhost:50051 list, then press Ctrl+C")

	<-ctx.Done()
	stop()

	log.Println("shutting down, waiting for in-flight RPCs")
	hs.Shutdown() // health reports NOT_SERVING, so no new calls are sent here

	// GracefulStop has no timeout, a long stream can keep it waiting forever.
	// Keep it under the time your platform gives you before SIGKILL.
	// Not: srv.Stop() right away, it cancels every running RPC.
	timer := time.AfterFunc(10*time.Second, srv.Stop)
	defer timer.Stop()

	srv.GracefulStop()

	log.Println("stopped")
}

// end
