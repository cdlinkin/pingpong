package messages

import (
	"encoding/json"
	"net/http"

	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/cdlinkin/pingpong/internal/httpx"
)

// ToggleReaction godoc
//
//	@Summary		Поставить или снять реакцию
//	@Description	Работает как переключатель: у одного пользователя на сообщении может быть только одна реакция. Та же эмодзи ещё раз снимает её, другая заменяет. Разрешены: 👍 ❤️ 😂 😮 😢 🔥. На удалённое сообщение реакцию поставить нельзя. В ответе актуальный список реакций сообщения; участники чата онлайн получают по WebSocket событие `{"type":"message.reactions","chat_id":13,"id":120,"reactions":[...]}`.
//	@Tags			messages
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"ID чата"
//	@Param			msgID	path		int					true	"ID сообщения"
//	@Param			request	body		reactionRequest		true	"Эмодзи"
//	@Success		200		{array}		reactionResponse	"Реакции сообщения после изменения"
//	@Failure		400		{string}	string				"invalid emoji"
//	@Failure		401		{string}	string				"invalid token"
//	@Failure		404		{string}	string				"chat not found / message not found (удалено или из другого чата)"
//	@Failure		500		{string}	string				"internal error"
//	@Router			/chats/{id}/messages/{msgID}/reactions [post]
func (h *Handler) ToggleReaction(rw http.ResponseWriter, r *http.Request) {
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

	var req reactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			rw,
			"invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	reactions, err := h.service.ToggleReaction(
		r.Context(),
		userID,
		chatID,
		msgID,
		req.Emoji,
	)
	if err != nil {
		h.writeError(rw, "toggle reaction", err, "invalid emoji")
		return
	}
	response := toReactions(reactions)

	h.broadcast(
		r.Context(),
		chatID,
		reactionEvent{
			Type:      "message.reactions",
			ChatID:    chatID,
			MsgID:     msgID,
			Reactions: response,
		})

	if err := httpx.WriteJSON(rw, http.StatusOK, response); err != nil {
		h.logger.Warn("toggle reaction: encode response", "err", err)
	}
}

func toReactions(rs []ReactionModel) []reactionResponse {
	out := make([]reactionResponse, 0)
	index := make(map[string]int)
	for _, r := range rs {
		i, ok := index[r.Emoji]
		if !ok {
			i = len(out)
			index[r.Emoji] = i
			out = append(out, reactionResponse{Emoji: r.Emoji})
		}
		out[i].UserIDs = append(out[i].UserIDs, r.UserID)
	}
	return out
}
