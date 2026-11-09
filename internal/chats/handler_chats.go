package chats

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/cdlinkin/pingpong/internal/httpx"
)

// CreateGroupChat godoc
//
//	@Summary		Создать групповой чат
//	@Description	Создаёт группу. Создатель автоматически становится участником с ролью `admin`.
//	@Tags			chats
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		createGroupRequest	true	"Название группы (1–50 символов)"
//	@Success		201		{object}	createChatResponse	"Группа создана"
//	@Failure		400		{string}	string				"Невалидный JSON или название"
//	@Failure		401		{string}	string				"invalid token"
//	@Failure		500		{string}	string				"internal error"
//	@Router			/chats/group [post]
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

	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			rw,
			"invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	id, err := h.service.CreateGroupChat(r.Context(), req.Title, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			http.Error(
				rw,
				err.Error(),
				http.StatusBadRequest,
			)
		default:
			h.logger.Error("create group chat", "err", err)
			http.Error(
				rw,
				"internal error",
				http.StatusInternalServerError,
			)
		}
		return
	}

	response := createChatResponse{
		ID: id,
	}

	if err := httpx.WriteJSON(rw, http.StatusCreated, response); err != nil {
		h.logger.Warn("create group chat: encode response", "err", err)
	}
}

// CreateDirectChat godoc
//
//	@Summary		Личный чат: найти или создать
//	@Description	Если личный чат с этим пользователем уже есть, возвращает его id со статусом 200. Иначе создаёт новый и отвечает 201. Чат с самим собой создать нельзя.
//	@Tags			chats
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		createDirectRequest	true	"id собеседника"
//	@Success		200		{object}	createChatResponse	"Чат уже существовал"
//	@Success		201		{object}	createChatResponse	"Чат создан"
//	@Failure		400		{string}	string				"Невалидный JSON или чат с самим собой"
//	@Failure		401		{string}	string				"invalid token"
//	@Failure		404		{string}	string				"user not found"
//	@Failure		500		{string}	string				"internal error"
//	@Router			/chats/direct [post]
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

	var req createDirectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			rw,
			"invalid JSON",
			http.StatusBadRequest,
		)
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
			h.logger.Error("create direct chat", "err", err)
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
	if err := httpx.WriteJSON(rw, status, response); err != nil {
		h.logger.Warn("create direct chat: encode response", "err", err)
	}
}

// ListUserChats godoc
//
//	@Summary		Мои чаты
//	@Description	Список чатов, в которых состоит текущий пользователь. У личных чатов `title` равен null, а в `peer_login` лежит логин собеседника. `unread_count` — сколько чужих неудалённых сообщений новее моей отметки прочтения (см. `POST /chats/{id}/read`).
//	@Tags			chats
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		chatResponse	"Список чатов (может быть пустым)"
//	@Failure		401	{string}	string			"invalid token"
//	@Failure		500	{string}	string			"internal error"
//	@Router			/chats [get]
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

	chats, err := h.service.ListUserChats(r.Context(), userID)
	if err != nil {
		h.logger.Error("list user chats", "err", err)
		http.Error(
			rw,
			"internal error",
			http.StatusInternalServerError,
		)
		return
	}

	response := make([]chatResponse, 0, len(chats))
	for _, c := range chats {
		response = append(response,
			chatResponse{
				ID:          c.ID,
				Title:       c.Title,
				Type:        c.Type,
				CreatedAt:   c.CreatedAt,
				Muted:       c.Muted,
				UnreadCount: c.UnreadCount,
				PeerLogin:   c.PeerLogin,
			},
		)
	}

	if err := httpx.WriteJSON(rw, http.StatusOK, response); err != nil {
		h.logger.Warn("list user chats: encode response", "err", err)
	}
}
