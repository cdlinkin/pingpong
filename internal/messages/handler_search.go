package messages

import (
	"net/http"

	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/cdlinkin/pingpong/internal/httpx"
)

// SearchMessages godoc
//
//	@Summary		Поиск по сообщениям
//	@Description	Ищет подстроку в сообщениях чата без учёта регистра. Возвращает до 50 последних совпадений.
//	@Tags			messages
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int			true	"ID чата"
//	@Param			q	query		string		true	"Текст для поиска (1–100 символов)"
//	@Success		200	{array}		msgResponse	"Найденные сообщения"
//	@Failure		400	{string}	string		"query must be 1-100 symbols"
//	@Failure		401	{string}	string		"invalid token"
//	@Failure		404	{string}	string		"chat not found"
//	@Failure		500	{string}	string		"internal error"
//	@Router			/chats/{id}/messages/search [get]
func (h *Handler) SearchMessages(rw http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query().Get("q")

	messages, err := h.service.SearchMessages(r.Context(), userID, chatID, q)
	if err != nil {
		h.writeError(rw, "search message", err, "query must be 1-100 symbols")
		return
	}

	response := make([]msgResponse, 0, len(messages))
	for _, m := range messages {
		response = append(response, toResponse(m))
	}

	if err := httpx.WriteJSON(rw, http.StatusOK, response); err != nil {
		h.logger.Warn("search message: encode response", "err", err)
	}
}
