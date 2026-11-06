package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"google.golang.org/grpc"

	pb "bookflow/gen/catalog"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://admin:secret@localhost:27017/?authSource=admin"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = ":50052"
	}

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		slog.Error("mongo connect", "err", err)
		os.Exit(1)
	}
	defer client.Disconnect(ctx)

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		slog.Error("mongo ping", "err", err)
		os.Exit(1)
	}
	slog.Info("connected to mongodb")

	store := NewStore(client)
	server := NewCatalogServer(store)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCatalogServiceServer(grpcServer, server)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		grpcServer.GracefulStop()
	}()

	log.Printf("Catalog gRPC listening on %s\n", port)
	grpcServer.Serve(lis)
}
