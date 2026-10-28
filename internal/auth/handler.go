package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
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

	rw.Header().Set("Content-Type", "application/json")

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
			log.Printf("register: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := registerResponse{
		ID: userID,
	}

	rw.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("register: encode response: %v", err)
		return
	}
}

// Login godoc
//
//	@Summary		Вход
//	@Description	Проверяет email и пароль и выдаёт JWT (HS256, живёт 24 часа). Токен передаётся в заголовке `Authorization: Bearer <token>`.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		loginRequest	true	"Email и пароль"
//	@Success		200		{object}	loginResponse	"JWT"
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

	rw.Header().Set("Content-Type", "application/json")

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
			log.Printf("login: %v", err)
			http.Error(rw, "internal error", http.StatusInternalServerError)
		}
		return
	}

	response := loginResponse{
		Token: token,
	}

	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("login: encode response: %v", err)
		return
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
	rw.Header().Set("Content-Type", "application/json")

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
			log.Printf("me: %v", err)
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

	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(response); err != nil {
		log.Printf("me: encode response: %v", err)
		return
	}
}
