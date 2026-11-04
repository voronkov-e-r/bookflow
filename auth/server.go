package main

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "bookflow/gen/auth"
)

type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	store     *Store
	jwtSecret []byte
}

func NewAuthServer(store *Store, jwtSecret string) *AuthServer {
	return &AuthServer{
		store:     store,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *AuthServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "email, password, name required")
	}
	if len(req.Password) < 6 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 6 chars")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "hash: %v", err)
	}

	id, err := s.store.CreateUser(ctx, req.Email, req.Name, string(hash))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create user: %v", err)
	}

	return &pb.RegisterResponse{UserId: id.String()}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := s.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	token, err := s.generateToken(user.ID.String())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "token: %v", err)
	}

	return &pb.LoginResponse{Token: token, UserId: user.ID.String()}, nil
}

func (s *AuthServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(req.Token, claims, func(t *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return &pb.ValidateTokenResponse{Ok: false}, nil
	}

	userID, _ := claims["sub"].(string)
	return &pb.ValidateTokenResponse{Ok: true, UserId: userID}, nil
}

func (s *AuthServer) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
