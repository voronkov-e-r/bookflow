package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrEmailExists = errors.New("email already exists")
)

type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) CreateUser(ctx context.Context, email, name, passwordHash string) (uuid.UUID, error) {
	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id
	`, email, passwordHash, name).Scan(&id)
	if err != nil {
		slog.Error("user insert failed", "err", err, "email", email)
		return uuid.Nil, err
	}
	return uuid.Parse(id)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	var idStr string
	err := s.db.QueryRow(ctx, `
		SELECT id, email, name, password_hash FROM users WHERE email = $1
	`, email).Scan(&idStr, &u.Email, &u.Name, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return User{}, err
	}
	u.ID = id
	return u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	var idStr string
	err := s.db.QueryRow(ctx, `
		SELECT id, email, name, password_hash FROM users WHERE id = $1
	`, id).Scan(&idStr, &u.Email, &u.Name, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	u.ID = id
	return u, nil
}
