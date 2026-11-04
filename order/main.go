package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	catalogpb "bookflow/gen/catalog"
	pb "bookflow/gen/order"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://default:secret@localhost:5432/bookflow?sslmode=disable")
	if err != nil {
		log.Fatalf("Postgres connect error: %v", err)
		os.Exit(1)
	}
	defer pool.Close()

	mqConn, err := amqp091.Dial("amqp://admin:secret@localhost:5672/")
	if err != nil {
		log.Fatalf("RabbitMQ connect error: %v", err)
		os.Exit(1)
	}
	defer mqConn.Close()

	mqChannal, err := mqConn.Channel()
	if err != nil {
		log.Fatalf("RabbitMQ channal connect error: %v", err)
		os.Exit(1)
	}
	defer mqChannal.Close()

	if err := mqChannal.ExchangeDeclare("orders", "fanout", true, false, false, false, nil); err != nil {
		log.Fatalf("RabbitMQ exchange declare error: %v", err)
		os.Exit(1)
	}

	catalogConn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Catalog connection error: %v", err)
		os.Exit(1)
	}
	defer catalogConn.Close()

	catalogClient := catalogpb.NewCatalogServiceClient(catalogConn)

	store := NewStore(pool)
	server := NewOrderServer(store, catalogClient, mqChannal)

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Listen service error: %v", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(grpcServer, server)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		grpcServer.GracefulStop()
	}()

	log.Println("Order gRPC listening on :50053")
	grpcServer.Serve(lis)
}
