package chats

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/cdlinkin/pingpong/internal/httpx"
)

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
	chatID, err := httpx.PathInt(r, "id")
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
			h.logger.Error("add member", "err", err)
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
	chatID, err := httpx.PathInt(r, "id")
	if err != nil {
		http.Error(
			rw,
			"invalid chat id",
			http.StatusBadRequest,
		)
		return
	}
	targetID, err := httpx.PathInt(r, "userID")
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
			h.logger.Error("remove member", "err", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}

// ListMembers godoc
//
//	@Summary		Участники чата
//	@Description	Список участников чата с логинами и ролями, отсортированный по логину. Если вы не участник чата, ответ 404.
//	@Tags			chats
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int				true	"ID чата"
//	@Success		200	{array}		memberResponse	"Участники"
//	@Failure		400	{string}	string			"invalid chat id"
//	@Failure		401	{string}	string			"invalid token"
//	@Failure		404	{string}	string			"chat not found"
//	@Failure		500	{string}	string			"internal error"
//	@Router			/chats/{id}/members [get]
func (h *Handler) ListMembers(rw http.ResponseWriter, r *http.Request) {
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

	members, err := h.service.ListMembers(r.Context(), userID, chatID)
	if err != nil {
		switch {
		case errors.Is(err, ErrChatNotFound):
			http.Error(rw, "chat not found", http.StatusNotFound)
		default:
			h.logger.Error("list members", "err", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := make([]memberResponse, 0, len(members))
	for _, m := range members {
		response = append(response,
			memberResponse{
				UserID: m.UserID,
				Login:  m.Login,
				Role:   m.Role,
			},
		)
	}

	if err := httpx.WriteJSON(rw, http.StatusOK, response); err != nil {
		h.logger.Warn("list members: encode response", "err", err)
	}
}
