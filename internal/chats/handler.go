package chats

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/cdlinkin/pingpong/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(
	service *Service,
) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) CreateGroupChat(rw http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(
			rw,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}
	rw.Header().Set("Content-Type", "application/json")

	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, "invalid JSON", http.StatusBadRequest)
		return
	}

	id, err := h.service.CreateGroupChat(r.Context(), req.Title, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, err.Error(), http.StatusBadRequest)
		default:
			log.Printf("create group chat: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := createChatResponse{
		ID: id,
	}

	rw.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("create group chat: encode response: %v", err)
		return
	}
}

func (h *Handler) CreateDirectChat(rw http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(
			rw,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}
	rw.Header().Set("Content-Type", "application/json")

	var req createDirectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, "invalid JSON", http.StatusBadRequest)
		return
	}

	id, created, err := h.service.CreateDirectChat(r.Context(), userID, req.UserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			http.Error(rw, "user not found", http.StatusNotFound)
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, err.Error(), http.StatusBadRequest)
		default:
			log.Printf("create direct chat: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := createChatResponse{
		ID: id,
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	rw.WriteHeader(status)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("direct chat: encode response: %v", err)
		return
	}
}

func (h *Handler) ListUserChats(rw http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(
			rw,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}
	rw.Header().Set("Content-Type", "application/json")

	chats, err := h.service.ListUserChats(r.Context(), userID)
	if err != nil {
		log.Printf("list user chats: %v", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	response := make([]chatResponse, 0, len(chats))
	for _, c := range chats {
		response = append(response,
			chatResponse{
				ID:        c.ID,
				Title:     c.Title,
				Type:      c.Type,
				CreatedAt: c.CreatedAt,
			},
		)
	}

	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("list user chats: encode response: %v", err)
		return
	}
}
