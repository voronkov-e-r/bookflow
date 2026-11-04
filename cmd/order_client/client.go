package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "bookflow/gen/auth"
	catalogpb "bookflow/gen/catalog"
	pb "bookflow/gen/order"
)

func main() {
	authConn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Connection error: %v", err)
	}
	defer authConn.Close()

	authClient := authpb.NewAuthServiceClient(authConn)

	catalogConn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Connection error: %v", err)
	}
	defer catalogConn.Close()

	catalogClient := catalogpb.NewCatalogServiceClient(catalogConn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orederConn, err := grpc.NewClient("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Connection error: %v", err)
	}
	defer orederConn.Close()

	orderClient := pb.NewOrderServiceClient(orederConn)

	bookReq := catalogpb.CreateBookRequest{
		Title:  "1984",
		Author: "Orwell",
		Year:   1988,
	}
	book, err := catalogClient.CreateBook(ctx, &bookReq)
	if err != nil {
		log.Fatalf("Create book error: %v", err)
	}

	user, err := authClient.Login(ctx, &authpb.LoginRequest{
		Email:    "mail@gmail.com",
		Password: "123456789",
	})
	if err != nil {
		log.Fatalf("Register error: %v", err)
	}

	order, err := orderClient.CreateOrder(ctx, &pb.CreateOrderRequest{
		UserId: user.UserId,
		BookId: book.Id,
	})
	if err != nil {
		log.Fatalf("Order create error: %v", err)
	}

	newBook, err := catalogClient.GetBook(ctx, &catalogpb.GetBookRequest{
		Id: book.Id,
	})
	if err != nil {
		log.Fatalf("Get book error: %v", err)
	}
	log.Printf("Get book with new status: %s\n", newBook.Status)

	_, err = orderClient.ReturnOrder(ctx, &pb.ReturnOrderRequest{
		Id: order.Id,
	})
	if err != nil {
		log.Fatalf("Return request error: %v", err)
	}

	lastBook, err := catalogClient.GetBook(ctx, &catalogpb.GetBookRequest{
		Id: book.Id,
	})
	if err != nil {
		log.Fatalf("Get book error: %v", err)
	}
	log.Printf("Get book with new status: %s\n", lastBook.Status)
}
