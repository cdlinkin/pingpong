package messages

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/cdlinkin/pingpong/internal/auth"
)

type Notifier interface {
	SendToUser(userID int, data []byte)
}

type Handler struct {
	service  *Service
	notifier Notifier
}

func NewHandler(service *Service, notifier Notifier) *Handler {
	return &Handler{
		service:  service,
		notifier: notifier,
	}
}

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

	msg, err := h.service.SendMessage(r.Context(), chatID, userID, req.Message, req.ReplyToID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, "message must be 1-500 symbols", http.StatusBadRequest)
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		case errors.Is(err, ErrMessageNotFound):
			http.Error(rw, "reply message not found", http.StatusNotFound)
		default:
			log.Printf("send message: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := toResponse(msg)

	rw.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("send message: encode response: %v", err)
		return
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
		response = append(response, toResponse(m))
	}

	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("list messages: encode response: %v", err)
		return
	}
}

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

	chatID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(rw, "invalid chat id", http.StatusBadRequest)
		return
	}

	q := r.URL.Query().Get("q")

	rw.Header().Set("Content-Type", "application/json")
	messages, err := h.service.SearchMessages(r.Context(), userID, chatID, q)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, "query must be 1-100 symbols", http.StatusBadRequest)
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		default:
			log.Printf("search message: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := make([]msgResponse, 0, len(messages))
	for _, m := range messages {
		response = append(response, toResponse(m))
	}

	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("search messages: encode response: %v", err)
		return
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
	chatID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(
			rw,
			"invalid chat id",
			http.StatusBadRequest,
		)
		return
	}

	msgID, err := strconv.Atoi(r.PathValue("msgID"))
	if err != nil {
		http.Error(
			rw,
			"invalid message id",
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

	msg, err := h.service.EditMessage(
		r.Context(),
		userID,
		chatID,
		msgID,
		req.Message,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, "message must be 1-500 symbols", http.StatusBadRequest)
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		case errors.Is(err, ErrMessageNotFound):
			http.Error(rw, "message not found", http.StatusNotFound)
		default:
			log.Printf("edit message: %v", err)
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

	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("edit message: encode response: %v", err)
		return
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
	chatID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(
			rw,
			"invalid chat id",
			http.StatusBadRequest,
		)
		return
	}

	msgID, err := strconv.Atoi(r.PathValue("msgID"))
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
		switch {
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		case errors.Is(err, ErrMessageNotFound):
			http.Error(rw, "message not found", http.StatusNotFound)
		default:
			log.Printf("delete message: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
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

	chatID, err := strconv.Atoi(r.PathValue("id"))
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
		switch {
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		case errors.Is(err, ErrMessageNotFound):
			http.Error(rw, "message not found", http.StatusNotFound)
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, "invalid input", http.StatusBadRequest)
		default:
			log.Printf("mark read message: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
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

func (h *Handler) broadcast(
	ctx context.Context,
	chatID int,
	event any,
) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("marshal event: %v", err)
		return
	}

	ids, err := h.service.ListMemberIDs(ctx, chatID)
	if err != nil {
		log.Printf("list member ids: %v", err)
		return
	}

	for _, memberID := range ids {
		h.notifier.SendToUser(memberID, data)
	}
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

	reads, err := h.service.ListReads(r.Context(), userID, chatID)
	if err != nil {
		switch {
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		default:
			log.Printf("list reads: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
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

	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("list reads: encode response: %v", err)
		return
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

	if m.ReplyToID != nil && m.ReplyUserID != nil && m.ReplyMessage != nil {
		response.ReplyTo = &replyResponse{
			ID:      *m.ReplyToID,
			UserID:  *m.ReplyUserID,
			Message: *m.ReplyMessage,
		}
	}
	return response
}
