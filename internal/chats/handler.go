package chats

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

func NewHandler(
	service *Service,
) *Handler {
	return &Handler{
		service: service,
	}
}

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
	rw.Header().Set("Content-Type", "application/json")

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
			log.Printf("create group chat: %v", err)
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

	rw.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("create group chat: encode response: %v", err)
		return
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
	rw.Header().Set("Content-Type", "application/json")

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
			log.Printf("create direct chat: %v", err)
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
	rw.WriteHeader(status)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("direct chat: encode response: %v", err)
		return
	}
}

// ListUserChats godoc
//
//	@Summary		Мои чаты
//	@Description	Список чатов, в которых состоит текущий пользователь. У личных чатов `title` равен null. `unread_count` — сколько чужих неудалённых сообщений новее моей отметки прочтения (см. `POST /chats/{id}/read`).
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
	rw.Header().Set("Content-Type", "application/json")

	chats, err := h.service.ListUserChats(r.Context(), userID)
	if err != nil {
		log.Printf("list user chats: %v", err)
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
				UnreadCount: c.UnreadCount,
			},
		)
	}

	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("list user chats: encode response: %v", err)
		return
	}
}

// AddMember godoc
//
//	@Summary		Добавить участника
//	@Description	Добавляет пользователя в групповой чат. Доступно только админу чата. Если вы не участник чата, ответ 404.
//	@Tags			chats
//	@Accept			json
//	@Security		BearerAuth
//	@Param			id		path	int					true	"ID чата"
//	@Param			request	body	addMemberRequest	true	"id добавляемого пользователя"
//	@Success		204		"Участник добавлен"
//	@Failure		400		{string}	string	"Невалидный id чата или JSON"
//	@Failure		401		{string}	string	"invalid token"
//	@Failure		403		{string}	string	"only admin can add members"
//	@Failure		404		{string}	string	"chat not found / user not found"
//	@Failure		409		{string}	string	"user is already a member"
//	@Failure		500		{string}	string	"internal error"
//	@Router			/chats/{id}/members [post]
func (h *Handler) AddMember(rw http.ResponseWriter, r *http.Request) {
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

	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			rw,
			"invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	if err = h.service.AddMember(r.Context(), userID, chatID, req.UserID); err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			http.Error(rw, "user not found", http.StatusNotFound)
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		case errors.Is(err, ErrForbidden):
			http.Error(rw, "only admin can add members", http.StatusForbidden)
		case errors.Is(err, ErrAlreadyMember):
			http.Error(rw, "user is already a member", http.StatusConflict)
		default:
			log.Printf("add member: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}

// RemoveMember godoc
//
//	@Summary		Удалить участника
//	@Description	Удаляет пользователя из чата. Доступно только админу чата, удалить самого себя нельзя.
//	@Tags			chats
//	@Security		BearerAuth
//	@Param			id		path	int	true	"ID чата"
//	@Param			userID	path	int	true	"ID удаляемого пользователя"
//	@Success		204		"Участник удалён"
//	@Failure		400		{string}	string	"Невалидный id или cannot remove yourself"
//	@Failure		401		{string}	string	"invalid token"
//	@Failure		403		{string}	string	"only admin can remove members"
//	@Failure		404		{string}	string	"chat not found / member not found"
//	@Failure		500		{string}	string	"internal error"
//	@Router			/chats/{id}/members/{userID} [delete]
func (h *Handler) RemoveMember(rw http.ResponseWriter, r *http.Request) {
	requesterID, ok := auth.UserIDFromContext(r.Context())
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
	targetID, err := strconv.Atoi(r.PathValue("userID"))
	if err != nil {
		http.Error(
			rw,
			"invalid user id",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.service.RemoveMember(r.Context(), requesterID, chatID, targetID); err != nil {
		switch {
		case errors.Is(err, ErrNotMember):
			http.Error(rw, "member not found", http.StatusNotFound)
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		case errors.Is(err, ErrForbidden):
			http.Error(rw, "only admin can remove members", http.StatusForbidden)
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, "cannot remove yourself", http.StatusBadRequest)
		default:
			log.Printf("remove member: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}
