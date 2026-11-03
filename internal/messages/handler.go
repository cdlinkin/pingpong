package messages

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type Notifier interface {
	SendToUser(userID int, data []byte)
}

type Handler struct {
	service  *Service
	notifier Notifier
	logger   *slog.Logger
}

func NewHandler(service *Service, notifier Notifier, logger *slog.Logger) *Handler {
	return &Handler{
		service:  service,
		notifier: notifier,
		logger:   logger,
	}
}

func (h *Handler) broadcast(
	ctx context.Context,
	chatID int,
	event any,
) {
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("broadcast: marshal event", "chat_id", chatID, "err", err)
		return
	}

	ids, err := h.service.ListMemberIDs(ctx, chatID)
	if err != nil {
		h.logger.Error("broadcast: list member ids", "chat_id", chatID, "err", err)
		return
	}

	for _, memberID := range ids {
		h.notifier.SendToUser(memberID, data)
	}
}

func toResponse(m MessageModel) msgResponse {
	response := msgResponse{
		ID:        m.ID,
		ChatID:    m.ChatID,
		UserID:    m.UserID,
		Message:   m.Message,
		CreatedAt: m.CreatedAt,
		EditedAt:  m.EditedAt,
		Deleted:   m.DeletedAt != nil,
	}

	if len(m.Reactions) > 0 {
		response.Reactions = toReactions(m.Reactions)
	}

	if m.ReplyToID != nil && m.ReplyUserID != nil && m.ReplyMessage != nil {
		response.ReplyTo = &replyResponse{
			ID:      *m.ReplyToID,
			UserID:  *m.ReplyUserID,
			Message: *m.ReplyMessage,
		}
	}
	return response
}

func (h *Handler) writeError(rw http.ResponseWriter, op string, err error, badRequest string) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		http.Error(rw, badRequest, http.StatusBadRequest)
	case errors.Is(err, ErrChatNotFound):
		http.Error(rw, "chat not found", http.StatusNotFound)
	case errors.Is(err, ErrMessageNotFound):
		http.Error(rw, "message not found", http.StatusNotFound)
	default:
		h.logger.Error(op, "err", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
	}
}
