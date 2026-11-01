package main

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "bookflow/gen/catalog"
)

type CatalogServer struct {
	pb.UnimplementedCatalogServiceServer
	store *Store
}

func NewCatalogServer(store *Store) *CatalogServer {
	return &CatalogServer{store: store}
}

func (s *CatalogServer) CreateBook(ctx context.Context, req *pb.CreateBookRequest) (*pb.Book, error) {
	if req.Title == "" || req.Author == "" {
		return nil, status.Error(codes.InvalidArgument, "title and author required")
	}

	book := pb.Book{
		Title:     req.Title,
		Author:    req.Author,
		Year:      req.Year,
		Status:    "available",
		Tags:      req.Tags,
		Publisher: req.Publisher,
	}

	return s.store.Create(ctx, &book)
}

func (s *CatalogServer) GetBook(ctx context.Context, req *pb.GetBookRequest) (*pb.Book, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}

	resp, err := s.store.Get(ctx, req.Id)
	if errors.Is(err, ErrorNotFound) {
		return nil, status.Error(codes.NotFound, "book not found")
	}
	return resp, err
}

func (s *CatalogServer) ListBooks(ctx context.Context, req *pb.ListBooksRequest) (*pb.ListBooksResponse, error) {
	resp, err := s.store.List(ctx, req.Author, req.Tag, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	listBooks := pb.ListBooksResponse{
		Books: resp,
		Total: int32(len(resp)),
	}
	return &listBooks, err
}

func (s *CatalogServer) UpdateBook(ctx context.Context, req *pb.UpdateBookRequest) (*pb.Book, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}

	set := bson.M{}
	if req.Author != nil {
		set["author"] = *req.Author
	}
	if req.Title != nil {
		set["title"] = *req.Title
	}
	if req.Year != nil {
		set["year"] = *req.Year
	}
	if req.Status != nil {
		set["status"] = *req.Status
	}
	if req.Tags != nil {
		set["tags"] = req.Tags
	}
	if req.Publisher != nil {
		set["publisher"] = req.Publisher
	}

	if len(set) == 0 {
		return s.GetBook(ctx, &pb.GetBookRequest{Id: req.Id})
	}
	b, err := s.store.Update(ctx, req.Id, set)
	if errors.Is(err, ErrorNotFound) {
		return nil, status.Error(codes.NotFound, "book not found")
	}
	return b, err
}

func (s *CatalogServer) DeleteBook(ctx context.Context, req *pb.DeleteBookRequest) (*pb.DeleteBookResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}

	err := s.store.Delete(ctx, req.Id)
	if errors.Is(err, ErrorNotFound) {
		return nil, status.Error(codes.NotFound, "book not found")
	}

	return &pb.DeleteBookResponse{Success: true}, err
}
