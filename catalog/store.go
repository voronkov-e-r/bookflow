package main

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	pb "bookflow/gen/catalog"
)

var ErrorNotFound error = errors.New("not found")

type Store struct {
	coll *mongo.Collection
}

func NewStore(client *mongo.Client) *Store {
	return &Store{client.Database("bookflow").Collection("books")}
}

func (s *Store) Create(ctx context.Context, b *pb.Book) (*pb.Book, error) {
	res, err := s.coll.InsertOne(ctx, b)
	if err != nil {
		return nil, err
	}
	b.Id = res.InsertedID.(bson.ObjectID).Hex()
	return b, nil
}

func (s *Store) Get(ctx context.Context, id string) (*pb.Book, error) {
	objId, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var b pb.Book
	err = s.coll.FindOne(ctx, bson.M{"_id": objId}).Decode(&b)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrorNotFound
	}

	return &b, err
}

func (s *Store) List(ctx context.Context, author, tag string, limit, offset int32) ([]*pb.Book, error) {
	filter := bson.M{}

	if author != "" {
		filter["author"] = author
	}
	if tag != "" {
		filter["tags"] = tag
	}

	opts := options.Find().SetSort(bson.M{"_id": -1})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}

	cursor, err := s.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var books []*pb.Book
	if err = cursor.All(ctx, &books); err != nil {
		return nil, err
	}
	return books, nil
}

func (s *Store) Update(ctx context.Context, id string, set bson.M) (*pb.Book, error) {
	objId, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var b pb.Book

	err = s.coll.FindOneAndUpdate(ctx, bson.M{"_id": objId}, bson.M{"$set": set}, opts).Decode(&b)

	if err == mongo.ErrNoDocuments {
		return nil, ErrorNotFound
	}
	return &b, err
}

func (s *Store) Delete(ctx context.Context, id string) error {
	objId, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	res, err := s.coll.DeleteOne(ctx, bson.M{"_id": objId})
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return ErrorNotFound
	}

	return nil
}
