package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "bookflow/gen/catalog"
)

func main() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Connection error: %v", err)
	}
	defer conn.Close()

	client := pb.NewCatalogServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	book, err := client.CreateBook(ctx, &pb.CreateBookRequest{
		Title:  "1984",
		Author: "Orwell",
		Year:   1949,
		Tags:   []string{"dystopia"},
	})
	if err != nil {
		log.Fatalf("Create error: %v", err)
	}
	id := book.Id
	log.Printf("Create response, id: %s, title: %s\n", book.Id, book.Title)

	book, err = client.GetBook(ctx, &pb.GetBookRequest{
		Id: id,
	})
	if err != nil {
		log.Fatalf("Get book error: %v", err)
	}
	log.Printf("Get book, id: %s, title: %s\n", id, book.Title)

	newTitle := "New Title"
	book, err = client.UpdateBook(ctx, &pb.UpdateBookRequest{
		Id:    id,
		Title: &newTitle,
	})
	if err != nil {
		log.Fatalf("Update book error: %v", err)
	}
	log.Printf("Update response, id: %s, title: %s\n", id, book.Title)

	ok, err := client.DeleteBook(ctx, &pb.DeleteBookRequest{
		Id: id,
	})
	if err != nil {
		log.Fatalf("Delete book error: %v", err)
	}
	log.Printf("Delete response, id: %s, is deleted: %t\n", id, ok.Success)
}
