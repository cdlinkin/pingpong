package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/cdlinkin/pingpong/internal/chats"
	"github.com/cdlinkin/pingpong/internal/messages"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Response struct {
	Status string `json:"status"`
}

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

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("godotenv load: %v", err)
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

	msgRepo := messages.NewRepository(pool)
	msgService := messages.NewService(msgRepo)
	msgHandler := messages.NewHandler(msgService)

	mux := http.NewServeMux()

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

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("error to run server: %v", err)
	}
}
