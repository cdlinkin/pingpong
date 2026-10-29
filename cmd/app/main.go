package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/cdlinkin/pingpong/docs"
	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/cdlinkin/pingpong/internal/chats"
	"github.com/cdlinkin/pingpong/internal/messages"
	"github.com/cdlinkin/pingpong/internal/ws"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger/v2"
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

	rw.Header().Set("Content-Type", "application/json")

	res := Response{
		Status: "OK",
	}
	rw.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(rw).Encode(res); err != nil {
		log.Printf("health: encode response: %v", err)
		return
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
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file, using environment variables: %v", err)
	}
	url := os.Getenv("DATABASE_URL")

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatalf("pgx pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("pool ping: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatalf("env no set")
	}

	repo := auth.NewRepository(pool)
	service := auth.NewService(repo, []byte(jwtSecret))
	handler := auth.NewHandler(service)
	middleware := auth.NewMiddleware([]byte(jwtSecret))

	chatsRepo := chats.NewRepository(pool)
	chatsService := chats.NewService(chatsRepo)
	chatsHandler := chats.NewHandler(chatsService)

	wsHub := ws.NewHub()

	msgRepo := messages.NewRepository(pool)
	msgService := messages.NewService(msgRepo)
	msgHandler := messages.NewHandler(msgService, wsHub)

	wsHandler := ws.NewHandler([]byte(jwtSecret), wsHub, msgService)

	mux := http.NewServeMux()

	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.PersistAuthorization(true),
	))

	mux.Handle("GET /me", middleware.Auth(http.HandlerFunc(handler.Me)))
	mux.HandleFunc("/login", handler.Login)
	mux.HandleFunc("/register", handler.Register)
	mux.HandleFunc("/health", health)
	mux.Handle("POST /chats/group", middleware.Auth(http.HandlerFunc(chatsHandler.CreateGroupChat)))
	mux.Handle("POST /chats/direct", middleware.Auth(http.HandlerFunc(chatsHandler.CreateDirectChat)))
	mux.Handle("GET /chats", middleware.Auth(http.HandlerFunc(chatsHandler.ListUserChats)))
	mux.Handle("POST /chats/{id}/members", middleware.Auth(http.HandlerFunc(chatsHandler.AddMember)))
	mux.Handle("DELETE /chats/{id}/members/{userID}", middleware.Auth(http.HandlerFunc(chatsHandler.RemoveMember)))
	mux.Handle("POST /chats/{id}/messages", middleware.Auth(http.HandlerFunc(msgHandler.SendMessage)))
	mux.Handle("GET /chats/{id}/messages", middleware.Auth(http.HandlerFunc(msgHandler.ListMessage)))
	mux.Handle("GET /chats/{id}/messages/search", middleware.Auth(http.HandlerFunc(msgHandler.SearchMessages)))
	mux.HandleFunc("GET /ws", wsHandler.ServeWS)
	mux.HandleFunc("GET /{$}", index)
	mux.Handle("PATCH /chats/{id}/messages/{msgID}", middleware.Auth(http.HandlerFunc(msgHandler.EditMessage)))
	mux.Handle("DELETE /chats/{id}/messages/{msgID}", middleware.Auth(http.HandlerFunc(msgHandler.DeleteMessage)))

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("error to run server: %v", err)
	}
}
