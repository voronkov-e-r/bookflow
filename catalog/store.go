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

	var raw []bson.M
	if err = cursor.All(ctx, &raw); err != nil {
		return nil, err
	}

	books := make([]*pb.Book, 0, len(raw))
	for _, r := range raw {
		b := &pb.Book{}

		if id, ok := r["_id"].(bson.ObjectID); ok {
			b.Id = id.Hex()
		}

		// Достаем строки
		if title, ok := r["title"].(string); ok {
			b.Title = title
		}
		if author, ok := r["author"].(string); ok {
			b.Author = author
		}
		if status, ok := r["status"].(string); ok {
			b.Status = status
		}

		if year, ok := r["year"].(int32); ok {
			b.Year = year
		} else if year64, ok := r["year"].(int64); ok {
			b.Year = int32(year64)
		}

		if tagsRaw, ok := r["tags"].(bson.A); ok {
			for _, t := range tagsRaw {
				if tStr, ok := t.(string); ok {
					b.Tags = append(b.Tags, tStr)
				}
			}
		}

		if pubRaw, ok := r["publisher"].(bson.M); ok {
			pub := &pb.Publisher{}
			if name, ok := pubRaw["name"].(string); ok {
				pub.Name = name
			}
			if country, ok := pubRaw["country"].(string); ok {
				pub.Country = country
			}
			b.Publisher = pub
		}

		books = append(books, b)
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
