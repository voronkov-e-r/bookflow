package main

import (
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authpb "bookflow/gen/auth"
	catalogpb "bookflow/gen/catalog"
	orderpb "bookflow/gen/order"
)

func (g *Gateway) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req authpb.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := g.auth.Register(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (g *Gateway) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authpb.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := g.auth.Login(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (g *Gateway) handleListBooks(w http.ResponseWriter, r *http.Request) {
	req := &catalogpb.ListBooksRequest{
		Author: r.URL.Query().Get("author"),
		Tag:    r.URL.Query().Get("tag"),
	}

	resp, err := g.catalog.ListBooks(r.Context(), req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (g *Gateway) handleGetBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	resp, err := g.catalog.GetBook(r.Context(), &catalogpb.GetBookRequest{Id: id})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (g *Gateway) handleCreateBook(w http.ResponseWriter, r *http.Request) {
	var req catalogpb.CreateBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := g.catalog.CreateBook(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (g *Gateway) handleUpdateBook(w http.ResponseWriter, r *http.Request) {
	var req catalogpb.UpdateBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := g.catalog.UpdateBook(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (g *Gateway) handleDeleteBook(w http.ResponseWriter, r *http.Request) {
	var req catalogpb.DeleteBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := g.catalog.DeleteBook(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (g *Gateway) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req orderpb.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := g.order.CreateOrder(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (g *Gateway) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	var req orderpb.GetOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := g.order.GetOrder(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (g *Gateway) handleListOrders(w http.ResponseWriter, r *http.Request) {
	var req orderpb.ListOrdersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	resp, err := g.order.ListOrders(r.Context(), &req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (g *Gateway) handleReturnOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	resp, err := g.order.ReturnOrder(r.Context(), &orderpb.ReturnOrderRequest{Id: id})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	httpStatus := http.StatusInternalServerError
	switch st.Code() {
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.AlreadyExists:
		httpStatus = http.StatusConflict
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	case codes.FailedPrecondition:
		httpStatus = http.StatusPreconditionFailed
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable
	}

	writeJSON(w, httpStatus, map[string]string{"error": st.Message()})
}
