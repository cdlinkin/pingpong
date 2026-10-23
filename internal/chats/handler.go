package chats

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

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

func (h *Handler) AddMember(rw http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(
			rw,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}
	chatID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(rw, "invalid chat id", http.StatusBadRequest)
		return
	}

	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err = h.service.AddMember(r.Context(), userID, chatID, req.UserID); err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			http.Error(rw, "user not found", http.StatusNotFound)
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		case errors.Is(err, ErrForbidden):
			http.Error(rw, "only admin can add members", http.StatusForbidden)
		case errors.Is(err, ErrAlreadyMember):
			http.Error(rw, "user is already a member", http.StatusConflict)
		default:
			log.Printf("add member: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveMember(rw http.ResponseWriter, r *http.Request) {
	requesterID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(
			rw,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}
	chatID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(rw, "invalid chat id", http.StatusBadRequest)
		return
	}
	targetID, err := strconv.Atoi(r.PathValue("userID"))
	if err != nil {
		http.Error(rw, "invalid user id", http.StatusBadRequest)
		return
	}

	if err := h.service.RemoveMember(r.Context(), requesterID, chatID, targetID); err != nil {
		switch {
		case errors.Is(err, ErrNotMember):
			http.Error(rw, "member not found", http.StatusNotFound)
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		case errors.Is(err, ErrForbidden):
			http.Error(rw, "only admin can remove members", http.StatusForbidden)
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, "cannot remove yourself", http.StatusBadRequest)
		default:
			log.Printf("remove member: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}
