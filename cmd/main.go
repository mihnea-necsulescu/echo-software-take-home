package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := getEnv("PORT", "8080")

	fireblocksBaseURL, ok := os.LookupEnv("FIREBLOCKS_BASE_URL")
	if !ok || fireblocksBaseURL == "" {
		log.Fatal("FIREBLOCKS_BASE_URL not set")
	}

	fireblocksAPIKey, ok := os.LookupEnv("FIREBLOCKS_API_KEY")
	if !ok || fireblocksAPIKey == "" {
		log.Fatal("FIREBLOCKS_API_KEY not set")
	}

	fireblocksSecretKey, ok := os.LookupEnv("FIREBLOCKS_SECRET_KEY")
	if !ok || fireblocksSecretKey == "" {
		log.Fatal("FIREBLOCKS_SECRET_KEY not set")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "OK"}`))
	})

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
