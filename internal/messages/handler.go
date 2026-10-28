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

// SendMessage godoc
//
//	@Summary		Отправить сообщение (HTTP)
//	@Description	Сохраняет сообщение в чате. Отправитель должен быть участником чата. Для доставки собеседникам в реальном времени используйте WebSocket `/ws`.
//	@Tags			messages
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int			true	"ID чата"
//	@Param			request	body		msgRequest	true	"Текст сообщения (1–500 символов)"
//	@Success		201		{object}	msgResponse	"Сообщение сохранено"
//	@Failure		400		{string}	string		"message must be 1-500 symbols"
//	@Failure		401		{string}	string		"invalid token"
//	@Failure		404		{string}	string		"chat not found"
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
		log.Printf("search messages: encode response: %v", err)
		return
	}
}
