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

	response := createGroupResponse{
		ID: id,
	}

	rw.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("create group chat: encode response: %v", err)
		return
	}
}
