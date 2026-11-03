package messages

import (
	"encoding/json"
	"net/http"

	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/cdlinkin/pingpong/internal/httpx"
)

// MarkRead godoc
//
//	@Summary		Отметить чат прочитанным
//	@Description	Сообщает, что пользователь дочитал чат до сообщения `message_id`. Отметка только растёт и не может уйти дальше последнего сообщения чата. Участники онлайн получают по WebSocket событие `{"type":"message.read","chat_id":13,"user_id":38,"message_id":120}` с итоговой отметкой.
//	@Tags			messages
//	@Accept			json
//	@Security		BearerAuth
//	@Param			id		path	int				true	"ID чата"
//	@Param			request	body	markReadRequest	true	"До какого сообщения прочитано"
//	@Success		204		"Отметка сохранена"
//	@Failure		400		{string}	string	"invalid JSON / invalid input"
//	@Failure		401		{string}	string	"invalid token"
//	@Failure		404		{string}	string	"chat not found"
//	@Failure		500		{string}	string	"internal error"
//	@Router			/chats/{id}/read [post]
func (h *Handler) MarkRead(rw http.ResponseWriter, r *http.Request) {
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

	var req markReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			rw,
			"invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	lastReadID, err := h.service.MarkRead(r.Context(), chatID, userID, req.MessageID)
	if err != nil {
		h.writeError(rw, "mark read message", err, "invalid input")
		return
	}

	h.broadcast(
		r.Context(),
		chatID,
		readEvent{
			Type:   "message.read",
			ChatID: chatID,
			UserID: userID,
			MsgID:  lastReadID,
		},
	)

	rw.WriteHeader(http.StatusNoContent)
}

// ListReads godoc
//
//	@Summary		Отметки прочтения участников
//	@Description	До какого сообщения дочитал каждый участник чата. Сообщение «прочитано», если у кого-то из остальных участников `last_read_message_id` больше или равен его id.
//	@Tags			messages
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int				true	"ID чата"
//	@Success		200	{array}		readResponse	"Отметки участников"
//	@Failure		400	{string}	string			"invalid chat id"
//	@Failure		401	{string}	string			"invalid token"
//	@Failure		404	{string}	string			"chat not found"
//	@Failure		500	{string}	string			"internal error"
//	@Router			/chats/{id}/reads [get]
func (h *Handler) ListReads(rw http.ResponseWriter, r *http.Request) {
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

	reads, err := h.service.ListReads(r.Context(), userID, chatID)
	if err != nil {
		h.writeError(rw, "list reads", err, "invalid input")
		return
	}

	response := make([]readResponse, 0, len(reads))
	for _, rm := range reads {
		response = append(response,
			readResponse{
				UserID:     rm.UserID,
				LastReadID: rm.LastReadID,
			})
	}

	if err := httpx.WriteJSON(rw, http.StatusOK, response); err != nil {
		h.logger.Warn("list reads: encode response", "err", err)
	}
}
