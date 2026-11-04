package main

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "bookflow/gen/order"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, userId, bookId, title string) (*pb.Order, error) {
	id, err := uuid.Parse(userId)
	if err != nil {
		return nil, err
	}

	var name string
	err = s.db.QueryRow(ctx, `SELECT name FROM users WHERE id = $1`, id).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var orderId uuid.UUID
	var status string
	var createdAt time.Time
	err = s.db.QueryRow(ctx, `INSERT INTO orders (user_id, book_id, book_title)
	VALUES ($1, $2, $3) RETURNING id, status, created_at
	`, userId, bookId, title).Scan(&orderId, &status, &createdAt)
	if err != nil {
		return nil, err
	}

	return &pb.Order{
		Id:        orderId.String(),
		UserId:    userId,
		BookId:    bookId,
		BookTitle: title,
		Status:    status,
		CreatedAt: createdAt.Unix(),
	}, nil
}

func (s *Store) Get(ctx context.Context, orderId string) (*pb.Order, error) {
	id, err := uuid.Parse(orderId)
	if err != nil {
		return nil, err
	}

	var bookId, bookTitle, status string
	var userId uuid.UUID
	var createdAt time.Time
	err = s.db.QueryRow(ctx, `SELECT user_id, book_id, book_title, status, created_at FROM orders WHERE id = $1`, id).Scan(&userId, &bookId, &bookTitle, &status, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &pb.Order{
		Id:        orderId,
		UserId:    userId.String(),
		BookId:    bookId,
		BookTitle: bookTitle,
		Status:    status,
		CreatedAt: createdAt.Unix(),
	}, nil
}

func (s *Store) List(ctx context.Context, userId string) ([]*pb.Order, error) {
	userUUID, err := uuid.Parse(userId)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `SELECT id, book_id, book_title, status, created_at FROM orders WHERE user_id = $1`, userUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*pb.Order
	for rows.Next() {
		var orderUUID uuid.UUID
		var bookId, bookTitle, status string
		var createdAt time.Time

		err = rows.Scan(&orderUUID, &bookId, &bookTitle, &status, &createdAt)
		if err != nil {
			return nil, err
		}

		orders = append(orders, &pb.Order{
			Id:        orderUUID.String(),
			UserId:    userId,
			BookId:    bookId,
			BookTitle: bookTitle,
			Status:    status,
			CreatedAt: createdAt.Unix(),
		})
	}
	return orders, nil
}

func (s *Store) UpdateStatus(ctx context.Context, id, status string) (*pb.Order, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}

	var o pb.Order
	var idStr string
	var createdAt time.Time

	err = s.db.QueryRow(ctx, `
		UPDATE orders SET status = $1 WHERE id = $2
		RETURNING id, user_id, book_id, book_title, status, created_at
	`, status, uid).Scan(&idStr, &o.UserId, &o.BookId, &o.BookTitle, &o.Status, &createdAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	o.Id = idStr
	o.CreatedAt = createdAt.Unix()
	return &o, nil
}
