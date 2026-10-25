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
