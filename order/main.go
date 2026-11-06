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

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://default:secret@localhost:5432/bookflow?sslmode=disable"
	}
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://admin:secret@localhost:5672/"
	}
	catalogAddr := os.Getenv("CATALOG_URL")
	if catalogAddr == "" {
		catalogAddr = "localhost:50052"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = ":50053"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Postgres connect error: %v", err)
		os.Exit(1)
	}
	defer pool.Close()

	mqConn, err := amqp091.Dial(rabbitURL)
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

	catalogConn, err := grpc.NewClient(catalogAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Catalog connection error: %v", err)
		os.Exit(1)
	}
	defer catalogConn.Close()

	catalogClient := catalogpb.NewCatalogServiceClient(catalogConn)

	store := NewStore(pool)
	server := NewOrderServer(store, catalogClient, mqChannal)

	lis, err := net.Listen("tcp", port)
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

	log.Printf("Order gRPC listening on %s\n", port)
	grpcServer.Serve(lis)
}
