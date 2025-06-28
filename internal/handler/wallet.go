package handler

import (
	"encoding/json"
	"firego-wallet-service/internal/fireblocks"
	"firego-wallet-service/internal/model"
	"log"
	"net/http"
)

type FireblocksClient interface {
	CreateVaultAccount(req fireblocks.CreateVaultAccountRequest) (*fireblocks.CreateVaultAccountResponse, int, error)
}

type WalletRepository interface {
	Create(wallet *model.Wallet) error
}

type WalletHandler struct {
	walletRepo       WalletRepository
	fireblocksClient FireblocksClient
}

func NewWalletHandler(walletRepo WalletRepository, fireblocksClient FireblocksClient) *WalletHandler {
	return &WalletHandler{
		walletRepo:       walletRepo,
		fireblocksClient: fireblocksClient,
	}
}

func (h *WalletHandler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Wallet name is required", http.StatusBadRequest)
		return
	}

	fbReq := fireblocks.CreateVaultAccountRequest{
		Name: req.Name,
	}

	fbResp, statusCode, err := h.fireblocksClient.CreateVaultAccount(fbReq)
	if err != nil {
		log.Printf("Failed to create Fireblocks vault account: %v", err)

		if statusCode >= 400 && statusCode < 500 {
			http.Error(w, "Invalid request", http.StatusBadRequest)
		} else {
			http.Error(w, "Service unavailable", http.StatusInternalServerError)
		}
		return
	}

	wallet := model.Wallet{
		Name:           req.Name,
		VaultAccountID: fbResp.ID,
	}
	err = h.walletRepo.Create(&wallet)
	if err != nil {
		log.Printf("Failed to create wallet: %v", err)
		http.Error(w, "Failed to create wallet", http.StatusInternalServerError)
		return
	}

	response := CreateWalletResponse{
		ID:             wallet.ID,
		Name:           fbResp.Name,
		VaultAccountID: fbResp.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err = json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
