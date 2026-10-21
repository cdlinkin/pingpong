package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

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

	mux := http.NewServeMux()

	mux.HandleFunc("/health", health)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("error to run server: %v", err)
	}
}
