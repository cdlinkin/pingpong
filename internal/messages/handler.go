package messages

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

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) SendMessage(rw http.ResponseWriter, r *http.Request) {
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
		http.Error(
			rw,
			"invalid chat id",
			http.StatusBadRequest,
		)
		return
	}

	rw.Header().Set("Content-Type", "application/json")

	var req msgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			rw,
			"invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	msg, err := h.service.SendMessage(r.Context(), chatID, userID, req.Message)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, "message must be 1-500 symbols", http.StatusBadRequest)
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		default:
			log.Printf("send message: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := msgResponse{
		ID:        msg.ID,
		ChatID:    msg.ChatID,
		UserID:    msg.UserID,
		Message:   msg.Message,
		CreatedAt: msg.CreatedAt,
	}

	rw.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("send message: encode response: %v", err)
		return
	}
}

func (h *Handler) ListMessage(rw http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(rw, "unauthorized", http.StatusUnauthorized)
		return
	}

	chatID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(rw, "invalid chat id", http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")

	var beforeID, limit int
	if s := r.URL.Query().Get("before"); s != "" {
		beforeID, err = strconv.Atoi(s)
		if err != nil {
			http.Error(rw, "invalid before", http.StatusBadRequest)
			return
		}
	}
	if s := r.URL.Query().Get("limit"); s != "" {
		limit, err = strconv.Atoi(s)
		if err != nil {
			http.Error(rw, "invalid limit", http.StatusBadRequest)
			return
		}
	}

	messages, err := h.service.ListMessage(r.Context(), userID, chatID, beforeID, limit)
	if err != nil {
		switch {
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		default:
			log.Printf("list message: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := make([]msgResponse, 0, len(messages))
	for _, m := range messages {
		response = append(response,
			msgResponse{
				ID:        m.ID,
				ChatID:    m.ChatID,
				UserID:    m.UserID,
				Message:   m.Message,
				CreatedAt: m.CreatedAt,
			})
	}

	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("list messages: encode response: %v", err)
		return
	}
}
