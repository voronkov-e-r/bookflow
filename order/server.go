package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"log/slog"

	"github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogpb "bookflow/gen/catalog"
	pb "bookflow/gen/order"
)

var (
	StatusAvailable string = "available"
	StatusBorrowed  string = "borrowed"
	StatusReturned  string = "returned"
)

type OrderServer struct {
	pb.UnimplementedOrderServiceServer
	store     *Store
	catalog   catalogpb.CatalogServiceClient
	mqChannel *amqp091.Channel
}

func NewOrderServer(store *Store, catalog catalogpb.CatalogServiceClient, mq *amqp091.Channel) *OrderServer {
	return &OrderServer{store: store, catalog: catalog, mqChannel: mq}
}

func (s *OrderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.Order, error) {
	book, err := s.catalog.GetBook(ctx, &catalogpb.GetBookRequest{Id: req.BookId})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Create error: %v", err)
	}

	if book.Status != "available" {
		return nil, status.Error(codes.Aborted, "Book is not available")
	}

	_, err = s.catalog.UpdateBook(ctx, &catalogpb.UpdateBookRequest{
		Id:     req.BookId,
		Status: &StatusBorrowed,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "Update book error")
	}

	order, err := s.store.Create(ctx, req.UserId, req.BookId, book.Title)
	if err != nil {
		_, err = s.catalog.UpdateBook(ctx, &catalogpb.UpdateBookRequest{
			Id:     req.BookId,
			Status: &StatusAvailable,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, "Book status rollback error")
		}
		return nil, status.Error(codes.Internal, "Create order error")
	}

	s.PublishOrderCreated(ctx, order)
	return order, nil
}

func (s *OrderServer) PublishOrderCreated(ctx context.Context, order *pb.Order) {
	body, err := json.Marshal(map[string]any{
		"event":      "order.created",
		"order_id":   order.Id,
		"user_id":    order.UserId,
		"book_id":    order.BookId,
		"book_title": order.BookTitle,
		"created_at": order.CreatedAt,
	})
	if err != nil {
		log.Printf("Marshal error: %v\n", err)
	}

	err = s.mqChannel.PublishWithContext(ctx, "orders", "", false, false, amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		Body:         body,
	})
	if err != nil {
		slog.Error("Publish \"created\" error", "error", err)
	}
}

func (s *OrderServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	order, err := s.store.Get(ctx, req.Id)
	if errors.Is(err, ErrNotFound) {
		return nil, status.Error(codes.NotFound, "Order not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "Get order error")
	}

	return order, nil
}

func (s *OrderServer) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	orders, err := s.store.List(ctx, req.UserId)
	if errors.Is(err, ErrNotFound) {
		return nil, status.Error(codes.NotFound, "Orders not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "Get list orders error")
	}
	return &pb.ListOrdersResponse{Orders: orders}, nil
}

func (s *OrderServer) ReturnOrder(ctx context.Context, req *pb.ReturnOrderRequest) (*pb.Order, error) {
	o, err := s.store.Get(ctx, req.Id)
	if errors.Is(err, ErrNotFound) {
		return nil, status.Error(codes.NotFound, "order not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	if o.Status == "returned" {
		return o, nil
	}

	available := "available"
	_, err = s.catalog.UpdateBook(ctx, &catalogpb.UpdateBookRequest{
		Id:     o.BookId,
		Status: &available,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "release book: %v", err)
	}

	updated, err := s.store.UpdateStatus(ctx, req.Id, "returned")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update order: %v", err)
	}
	return updated, nil
}
