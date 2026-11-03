package messages

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/cdlinkin/pingpong/internal/httpx"
)

// SendMessage godoc
//
//	@Summary		Отправить сообщение (HTTP)
//	@Description	Сохраняет сообщение в чате. Отправитель должен быть участником чата. Чтобы ответить на сообщение, передайте `reply_to_id` (сообщение должно быть в этом же чате), тогда в ответе появится цитата `reply_to`. Для доставки собеседникам в реальном времени используйте WebSocket `/ws`.
//	@Tags			messages
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int			true	"ID чата"
//	@Param			request	body		msgRequest	true	"Текст сообщения (1–500 символов) и, если это ответ, reply_to_id"
//	@Success		201		{object}	msgResponse	"Сообщение сохранено"
//	@Failure		400		{string}	string		"message must be 1-500 symbols"
//	@Failure		401		{string}	string		"invalid token"
//	@Failure		404		{string}	string		"chat not found / reply message not found"
//	@Failure		500		{string}	string		"internal error"
//	@Router			/chats/{id}/messages [post]
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
	chatID, err := httpx.PathInt(r, "id")
	if err != nil {
		http.Error(
			rw,
			"invalid chat id",
			http.StatusBadRequest,
		)
		return
	}

	var req msgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			rw,
			"invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	msg, err := h.service.SendMessage(r.Context(), chatID, userID, req.Message, req.ReplyToID)
	if err != nil {
		if errors.Is(err, ErrMessageNotFound) {
			http.Error(rw, "reply message not found", http.StatusNotFound)
			return
		}
		h.writeError(rw, "send message", err, "message must be 1-500 symbols")
		return
	}

	response := toResponse(msg)

	if err := httpx.WriteJSON(rw, http.StatusCreated, response); err != nil {
		h.logger.Warn("send message: encode response", "err", err)
	}
}

// ListMessage godoc
//
//	@Summary		История сообщений
//	@Description	Курсорная пагинация от новых к старым. Для следующей страницы передайте в `before` id самого старого сообщения из текущей.
//	@Tags			messages
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int			true	"ID чата"
//	@Param			before	query		int			false	"Вернуть сообщения с id меньше этого"
//	@Param			limit	query		int			false	"Размер страницы: по умолчанию 50, максимум 100"	minimum(1)	maximum(100)
//	@Success		200		{array}		msgResponse	"Страница сообщений"
//	@Failure		400		{string}	string		"invalid before / invalid limit"
//	@Failure		401		{string}	string		"invalid token"
//	@Failure		404		{string}	string		"chat not found"
//	@Failure		500		{string}	string		"internal error"
//	@Router			/chats/{id}/messages [get]
func (h *Handler) ListMessage(rw http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(rw, "unauthorized", http.StatusUnauthorized)
		return
	}

	chatID, err := httpx.PathInt(r, "id")
	if err != nil {
		http.Error(rw, "invalid chat id", http.StatusBadRequest)
		return
	}

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
		h.writeError(rw, "list message", err, "invalid input")
		return
	}

	response := make([]msgResponse, 0, len(messages))
	for _, m := range messages {
		response = append(response, toResponse(m))
	}

	if err := httpx.WriteJSON(rw, http.StatusOK, response); err != nil {
		h.logger.Warn("list message: encode response", "err", err)
	}
}

// EditMessage godoc
//
//	@Summary		Изменить сообщение
//	@Description	Меняет текст своего сообщения и ставит `edited_at`. Изменить можно только своё и не удалённое сообщение. Участники чата онлайн получают по WebSocket событие `{"type":"message.edited","message":{...}}`.
//	@Tags			messages
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int			true	"ID чата"
//	@Param			msgID	path		int			true	"ID сообщения"
//	@Param			request	body		msgRequest	true	"Новый текст (1–500 символов)"
//	@Success		200		{object}	msgResponse	"Сообщение изменено"
//	@Failure		400		{string}	string		"message must be 1-500 symbols"
//	@Failure		401		{string}	string		"invalid token"
//	@Failure		404		{string}	string		"chat not found / message not found (чужое, удалённое или не существует)"
//	@Failure		500		{string}	string		"internal error"
//	@Router			/chats/{id}/messages/{msgID} [patch]
func (h *Handler) EditMessage(rw http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(
			rw,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}
	chatID, err := httpx.PathInt(r, "id")
	if err != nil {
		http.Error(
			rw,
			"invalid chat id",
			http.StatusBadRequest,
		)
		return
	}

	msgID, err := httpx.PathInt(r, "msgID")
	if err != nil {
		http.Error(
			rw,
			"invalid message id",
			http.StatusBadRequest,
		)
		return
	}

	var req msgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			rw,
			"invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	msg, err := h.service.EditMessage(
		r.Context(),
		userID,
		chatID,
		msgID,
		req.Message,
	)
	if err != nil {
		h.writeError(rw, "edit message", err, "message must be 1-500 symbols")
		return
	}

	response := msgResponse{
		ID:        msg.ID,
		ChatID:    msg.ChatID,
		UserID:    msg.UserID,
		Message:   msg.Message,
		CreatedAt: msg.CreatedAt,
		EditedAt:  msg.EditedAt,
	}

	h.broadcast(
		r.Context(),
		chatID,
		editedEvent{
			Type:    "message.edited",
			Message: response,
		},
	)

	if err := httpx.WriteJSON(rw, http.StatusOK, response); err != nil {
		h.logger.Warn("edit message: encode response", "err", err)
	}
}

// DeleteMessage godoc
//
//	@Summary		Удалить сообщение
//	@Description	Мягкое удаление своего сообщения: строка остаётся в базе с `deleted_at`, а в истории сообщение приходит с `"deleted": true` и пустым текстом. В поиск удалённые не попадают. Участники чата онлайн получают по WebSocket событие `{"type":"message.deleted","chat_id":13,"id":120}`.
//	@Tags			messages
//	@Security		BearerAuth
//	@Param			id		path	int	true	"ID чата"
//	@Param			msgID	path	int	true	"ID сообщения"
//	@Success		204		"Сообщение удалено"
//	@Failure		400		{string}	string	"invalid message id"
//	@Failure		401		{string}	string	"invalid token"
//	@Failure		404		{string}	string	"chat not found / message not found (чужое, уже удалённое или не существует)"
//	@Failure		500		{string}	string	"internal error"
//	@Router			/chats/{id}/messages/{msgID} [delete]
func (h *Handler) DeleteMessage(rw http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(
			rw,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}
	chatID, err := httpx.PathInt(r, "id")
	if err != nil {
		http.Error(
			rw,
			"invalid chat id",
			http.StatusBadRequest,
		)
		return
	}

	msgID, err := httpx.PathInt(r, "msgID")
	if err != nil {
		http.Error(
			rw,
			"invalid message id",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.service.DeleteMessage(
		r.Context(),
		userID,
		chatID,
		msgID,
	); err != nil {
		h.writeError(rw, "delete message", err, "invalid input")
		return
	}

	h.broadcast(
		r.Context(),
		chatID,
		deletedEvent{
			Type:   "message.deleted",
			ChatID: chatID,
			MsgID:  msgID,
		},
	)

	rw.WriteHeader(http.StatusNoContent)
}
