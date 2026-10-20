package main

import (
	"encoding/json"
	"log"
	"net/http"
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
	mux := http.NewServeMux()

	mux.HandleFunc("/health", health)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("error to run server: %v", err)
		return
	}
}
