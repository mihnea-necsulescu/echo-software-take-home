package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"os"
	"time"
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

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		uri := "/v1/vault/accounts_paged"
		nonce := uuid.New().String()
		now := time.Now().Unix()
		h := sha256.New()
		h.Write(nil)
		hashed := h.Sum(nil)

		claims := jwt.MapClaims{
			"uri":      uri,
			"nonce":    nonce,
			"iat":      now,
			"exp":      now + 30,
			"sub":      fireblocksAPIKey,
			"bodyHash": hex.EncodeToString(hashed),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tokenString, err := token.SignedString(fireblocksPrivateKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to sign token: %v", err), http.StatusInternalServerError)
			return
		}

		fullURL := fireblocksBaseURL + uri
		req, _ := http.NewRequest("GET", fullURL, nil)
		req.Header.Set("X-API-Key", fireblocksAPIKey)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("API call failed: %v", err), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
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
