package main

import (
	"firego-wallet-service/internal/fireblocks"
	"firego-wallet-service/internal/handlers"
	"github.com/golang-jwt/jwt/v5"
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

	fireblocksSecretKeyPath, ok := os.LookupEnv("FIREBLOCKS_SECRET_KEY_PATH")
	if !ok || fireblocksSecretKeyPath == "" {
		log.Fatal("FIREBLOCKS_SECRET_KEY_PATH not set")
	}
	fireblocksPrivateKeyBytes, err := os.ReadFile(fireblocksSecretKeyPath)
	if err != nil {
		log.Fatalf("error reading private key from %s: %v", fireblocksSecretKeyPath, err)
	}
	fireblocksPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEM(fireblocksPrivateKeyBytes)
	if err != nil {
		log.Fatalf("error parsing RSA private key: %v", err)
	}

	mux := http.NewServeMux()

	fireblocksClient := fireblocks.NewClient(fireblocksBaseURL, fireblocksAPIKey, fireblocksPrivateKey)
	walletHandler := handlers.NewWalletHandler(fireblocksClient)

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		resp, _ := fireblocksClient.GetAccountsPaged()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(resp)
	})

	mux.HandleFunc("/wallets", walletHandler.CreateWallet)

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
