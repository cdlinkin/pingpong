package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cdlinkin/pingpong/internal/httpx"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Register godoc
//
//	@Summary		Регистрация
//	@Description	Создаёт нового пользователя. Логин 3–25 символов, корректный email, пароль от 8 до 72 байт (ограничение bcrypt).
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		registerRequest		true	"Данные нового пользователя"
//	@Success		201		{object}	registerResponse	"Пользователь создан"
//	@Failure		400		{string}	string				"Невалидный JSON или данные"
//	@Failure		405		{string}	string				"method not allowed"
//	@Failure		409		{string}	string				"user already exists"
//	@Failure		500		{string}	string				"internal error"
//	@Router			/register [post]
func (h *Handler) Register(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, "invalid JSON", http.StatusBadRequest)
		return
	}

	userID, err := h.service.Register(r.Context(), req.Login, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrUserExists):
			http.Error(rw, "user already exists", http.StatusConflict)
		default:
			h.logger.Error("register", "err", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := registerResponse{
		ID: userID,
	}

	if err := httpx.WriteJSON(rw, http.StatusCreated, response); err != nil {
		h.logger.Warn("register: encode response", "err", err)
	}
}

// Login godoc
//
//	@Summary		Вход
//	@Description	Проверяет email и пароль и выдаёт пару токенов. `token` — access JWT (HS256, живёт 15 минут), передаётся в заголовке `Authorization: Bearer <token>`. `refresh_token` живёт 30 дней и нужен, чтобы получить новую пару через `POST /refresh`.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		loginRequest	true	"Email и пароль"
//	@Success		200		{object}	loginResponse	"Пара токенов"
//	@Failure		400		{string}	string			"Невалидный JSON или данные"
//	@Failure		401		{string}	string			"invalid email or password"
//	@Failure		405		{string}	string			"method not allowed"
//	@Failure		500		{string}	string			"internal error"
//	@Router			/login [post]
func (h *Handler) Login(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, "invalid JSON", http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			http.Error(rw, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrInvalidCredentials):
			http.Error(rw, "invalid email or password", http.StatusUnauthorized)
		default:
			h.logger.Error("login", "err", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := loginResponse{
		Token:        token.Access,
		RefreshToken: token.Refresh,
	}

	if err := httpx.WriteJSON(rw, http.StatusOK, response); err != nil {
		h.logger.Warn("login: encode response", "err", err)
	}
}

// Me godoc
//
//	@Summary		Текущий пользователь
//	@Description	Возвращает профиль владельца токена.
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	meResponse	"Профиль"
//	@Failure		401	{string}	string		"invalid token"
//	@Failure		404	{string}	string		"user not found"
//	@Failure		500	{string}	string		"internal error"
//	@Router			/me [get]
func (h *Handler) Me(rw http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(
			rw,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	user, err := h.service.Me(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			http.Error(rw, "user not found", http.StatusNotFound)
		default:
			h.logger.Error("me", "err", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := meResponse{
		ID:        user.ID,
		Login:     user.Login,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	if err := httpx.WriteJSON(rw, http.StatusOK, response); err != nil {
		h.logger.Warn("me: encode response", "err", err)
	}
}

// Refresh godoc
//
//	@Summary		Обновление токенов
//	@Description	Меняет refresh-токен на новую пару токенов. Старый refresh-токен после этого перестаёт работать (ротация).
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		refreshRequest	true	"Refresh-токен из /login или /refresh"
//	@Success		200		{object}	loginResponse	"Новая пара токенов"
//	@Failure		400		{string}	string			"invalid JSON"
//	@Failure		401		{string}	string			"invalid refresh token"
//	@Failure		429		{string}	string			"too many requests"
//	@Failure		500		{string}	string			"internal error"
//	@Router			/refresh [post]
func (h *Handler) Refresh(rw http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			rw,
			"invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	tokens, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			http.Error(rw, "invalid refresh token", http.StatusUnauthorized)
			return
		}
		h.logger.Error("refresh", "err", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	response := loginResponse{
		Token:        tokens.Access,
		RefreshToken: tokens.Refresh,
	}

	if err := httpx.WriteJSON(rw, http.StatusOK, response); err != nil {
		h.logger.Warn("refresh: encode response", "err", err)
	}
}

// Logout godoc
//
//	@Summary		Выход
//	@Description	Удаляет refresh-токен. Access-токен не нужен. Повторный выход тоже возвращает 204.
//	@Tags			auth
//	@Accept			json
//	@Param			request	body	refreshRequest	true	"Refresh-токен"
//	@Success		204		"Токен удалён"
//	@Failure		400		{string}	string	"invalid JSON"
//	@Failure		429		{string}	string	"too many requests"
//	@Failure		500		{string}	string	"internal error"
//	@Router			/logout [post]
func (h *Handler) Logout(rw http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			rw,
			"invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.service.Logout(r.Context(), req.RefreshToken); err != nil {
		h.logger.Error("logout", "err", err)
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}
