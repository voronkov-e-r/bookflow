package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "bookflow/gen/auth"
	catalogpb "bookflow/gen/catalog"
	orderpb "bookflow/gen/order"
)

type Gateway struct {
	auth    authpb.AuthServiceClient
	catalog catalogpb.CatalogServiceClient
	order   orderpb.OrderServiceClient
}

func main() {
	authConn, _ := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer authConn.Close()

	catalogConn, _ := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer catalogConn.Close()

	orderConn, _ := grpc.NewClient("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer orderConn.Close()

	gw := &Gateway{
		auth:    authpb.NewAuthServiceClient(authConn),
		catalog: catalogpb.NewCatalogServiceClient(catalogConn),
		order:   orderpb.NewOrderServiceClient(orderConn),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/register", gw.handleRegister)
	mux.HandleFunc("POST /api/auth/login", gw.handleLogin)

	mux.HandleFunc("GET /api/books", gw.handleListBooks)
	mux.HandleFunc("GET /api/books/{id}", gw.handleGetBook)
	mux.HandleFunc("POST /api/books", gw.handleCreateBook)
	mux.HandleFunc("PUT /api/books/{id}", gw.handleUpdateBook)
	mux.HandleFunc("DELETE /api/books/{id}", gw.handleDeleteBook)

	mux.HandleFunc("POST /api/orders", gw.handleCreateOrder)
	mux.HandleFunc("GET /api/orders/{id}", gw.handleGetOrder)
	mux.HandleFunc("GET /api/orders", gw.handleListOrders)
	mux.HandleFunc("POST /api/orders/{id}/return", gw.handleReturnOrder)

	httpSrv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("gateway starting", "port", "8080")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpSrv.Shutdown(ctx)
}
