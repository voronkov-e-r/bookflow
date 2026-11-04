package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "bookflow/gen/auth"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewAuthServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reg, err := client.Register(ctx, &pb.RegisterRequest{
		Email:    "test@example.com",
		Password: "secret123",
		Name:     "Test User",
	})
	if err != nil {
		log.Printf("Register error (maybe exists): %v", err)
	} else {
		log.Printf("Registered: user_id=%s", reg.UserId)
	}

	login, err := client.Login(ctx, &pb.LoginRequest{
		Email:    "test@example.com",
		Password: "secret123",
	})
	if err != nil {
		log.Fatalf("Login error: %v", err)
	}
	log.Printf("Logged in: token=%s...", login.Token[:20])

	val, err := client.ValidateToken(ctx, &pb.ValidateTokenRequest{Token: login.Token})
	if err != nil {
		log.Fatalf("Validate error: %v", err)
	}
	log.Printf("Valid=%v, user_id=%s", val.Ok, val.UserId)
}
