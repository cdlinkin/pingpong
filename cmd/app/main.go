package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/cdlinkin/pingpong/docs"
	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/cdlinkin/pingpong/internal/chats"
	"github.com/cdlinkin/pingpong/internal/config"
	"github.com/cdlinkin/pingpong/internal/httpx"
	"github.com/cdlinkin/pingpong/internal/logging"
	"github.com/cdlinkin/pingpong/internal/messages"
	"github.com/cdlinkin/pingpong/internal/ratelimit"
	"github.com/cdlinkin/pingpong/internal/server"
	"github.com/cdlinkin/pingpong/internal/ws"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"golang.org/x/time/rate"
)

type Response struct {
	Status string `json:"status" example:"OK"`
} //@name HealthResponse

func index(rw http.ResponseWriter, r *http.Request) {
	http.ServeFile(rw, r, "web/index.html")
}

// health godoc
//
//	@Summary		Проверка состояния
//	@Description	Отвечает `{"status":"OK"}`, если сервер запущен.
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	Response	"Сервер работает"
//	@Router			/health [get]
func health(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	res := Response{
		Status: "OK",
	}

	if err := httpx.WriteJSON(rw, http.StatusOK, res); err != nil {
		slog.Warn("health: encode response", "err", err)
	}
}

//	@title			PingPong API
//	@version		1.0
//	@description	REST API веб-мессенджера PingPong: регистрация и JWT-авторизация, личные и групповые чаты, сообщения с курсорной пагинацией и поиском, доставка в реальном времени через WebSocket.
//	@description
//	@description	**Как авторизоваться:** выполните `POST /login`, нажмите **Authorize** и вставьте `Bearer <token>`.
//	@description
//	@description	**Ошибки** возвращаются обычным текстом (`text/plain`), например `chat not found`. Если вы не участник чата, API отвечает 404, чтобы не раскрывать, что чат существует.
//	@host			localhost:8080
//	@BasePath		/
//	@schemes		http

//	@tag.name			system
//	@tag.description	Служебные эндпоинты
//	@tag.name			auth
//	@tag.description	Регистрация, вход и профиль
//	@tag.name			chats
//	@tag.description	Личные и групповые чаты, участники
//	@tag.name			messages
//	@tag.description	Отправка, история и поиск сообщений
//	@tag.name			realtime
//	@tag.description	WebSocket

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				JWT из POST /login. Формат: `Bearer <token>`
func main() {
	cfg, cfgErr := config.Load()

	logger := logging.New(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)

	if !cfg.EnvFile {
		logger.Info("no .env file, using environment variables")
	}
	if cfgErr != nil {
		logger.Error("config", "err", cfgErr)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("pgx pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("pool ping", "err", err)
		os.Exit(1)
	}

	jwtSecret := []byte(cfg.JWTSecret)

	repo := auth.NewRepository(pool)
	service := auth.NewService(repo, jwtSecret)
	handler := auth.NewHandler(service, logger)
	middleware := auth.NewMiddleware(jwtSecret)

	chatsRepo := chats.NewRepository(pool)
	chatsService := chats.NewService(chatsRepo)
	chatsHandler := chats.NewHandler(chatsService, logger)

	wsHub := ws.NewHub()

	msgRepo := messages.NewRepository(pool)
	msgService := messages.NewService(msgRepo)
	msgHandler := messages.NewHandler(msgService, wsHub, logger)

	mux := http.NewServeMux()

	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.PersistAuthorization(true),
	))

	authLimit := ratelimit.New(rate.Every(12*time.Second), 5)
	userLimit := ratelimit.New(10, 20)
	wsLimit := ratelimit.New(5, 10)

	wsHandler := ws.NewHandler(jwtSecret, wsHub, msgService, wsLimit, logger)

	protected := func(h http.HandlerFunc) http.Handler {
		return middleware.Auth(userLimit.ByUser(h))
	}

	mux.Handle("GET /me", protected(handler.Me))
	mux.Handle("/login", authLimit.ByIP(http.HandlerFunc(handler.Login)))
	mux.Handle("/register", authLimit.ByIP(http.HandlerFunc(handler.Register)))
	mux.HandleFunc("/health", health)
	mux.Handle("POST /chats/group", protected(chatsHandler.CreateGroupChat))
	mux.Handle("POST /chats/direct", protected(chatsHandler.CreateDirectChat))
	mux.Handle("GET /chats", protected(chatsHandler.ListUserChats))
	mux.Handle("GET /chats/{id}/members", protected(chatsHandler.ListMembers))
	mux.Handle("POST /chats/{id}/members", protected(chatsHandler.AddMember))
	mux.Handle("DELETE /chats/{id}/members/{userID}", protected(chatsHandler.RemoveMember))
	mux.Handle("POST /chats/{id}/messages", protected(msgHandler.SendMessage))
	mux.Handle("GET /chats/{id}/messages", protected(msgHandler.ListMessage))
	mux.Handle("GET /chats/{id}/messages/search", protected(msgHandler.SearchMessages))
	mux.HandleFunc("GET /ws", wsHandler.ServeWS)
	mux.HandleFunc("GET /{$}", index)
	mux.Handle("PATCH /chats/{id}/messages/{msgID}", protected(msgHandler.EditMessage))
	mux.Handle("DELETE /chats/{id}/messages/{msgID}", protected(msgHandler.DeleteMessage))
	mux.Handle("POST /chats/{id}/read", protected(msgHandler.MarkRead))
	mux.Handle("GET /chats/{id}/reads", protected(msgHandler.ListReads))
	mux.Handle("POST /chats/{id}/messages/{msgID}/reactions", protected(msgHandler.ToggleReaction))

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           logging.Middleware(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	srv.RegisterOnShutdown(wsHub.CloseAll)

	if err := server.Run(
		ctx,
		srv,
		10*time.Second,
		logger,
	); err != nil {
		logger.Error("server", "err", err)
	}
}
