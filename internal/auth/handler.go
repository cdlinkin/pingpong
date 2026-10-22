package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Register(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rw.Header().Set("Content-Type", "application/json")

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, "invalid JSON", http.StatusBadRequest)
		return
	}

	userID, err := h.service.Register(r.Context(), req.Login, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrUserExists):
			http.Error(rw, "user already exists", http.StatusConflict)
		default:
			log.Printf("register: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := registerResponse{
		ID: userID,
	}

	rw.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("register: encode response: %v", err)
		return
	}
}

func (h *Handler) Login(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rw.Header().Set("Content-Type", "application/json")

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, "invalid JSON", http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrInvalidCredentials):
			http.Error(rw, "invalid email or password", http.StatusUnauthorized)
		default:
			log.Printf("login: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := loginResponse{
		Token: token,
	}

	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("login: encode response: %v", err)
		return
	}
}

func (h *Handler) Me(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")

	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(
			rw,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	user, err := h.service.Me(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			http.Error(rw, "user not found", http.StatusNotFound)
		default:
			log.Printf("me: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := meResponse{
		ID:        user.ID,
		Login:     user.Login,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("me: encode response: %v", err)
		return
	}
}
