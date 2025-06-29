package handler

import (
	"encoding/json"
	"errors"
	"firego-wallet-service/internal/fireblocks"
	"firego-wallet-service/internal/model"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
)

type FireblocksClient interface {
	CreateVaultAccount(req fireblocks.CreateVaultAccountRequest) (*fireblocks.CreateVaultAccountResponse, int, error)
	GetVaultAccountAssetBalance(vaultAccountID, assetID string) (*fireblocks.GetVaultAccountAssetBalanceResponse, int, error)
	GetVaultAccountAssetAddresses(vaultAccountID, assetID string) (*fireblocks.GetVaultAccountAssetAddressesResponse, int, error)
	CreateTransaction(req fireblocks.CreateTransactionRequest) (*fireblocks.CreateTransactionResponse, int, error)
}

type WalletRepository interface {
	Create(wallet *model.Wallet) error
	GetByID(id string) (*model.Wallet, error)
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

func (h *WalletHandler) GetWalletBalance(w http.ResponseWriter, r *http.Request) {
	walletID := r.PathValue("walletId")
	assetID := r.PathValue("assetId")

	if walletID == "" || assetID == "" {
		http.Error(w, "Wallet ID and Asset ID are required", http.StatusBadRequest)
		return
	}

	wallet, err := h.walletRepo.GetByID(walletID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Wallet not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to get wallet: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	fbResp, statusCode, err := h.fireblocksClient.GetVaultAccountAssetBalance(wallet.VaultAccountID, assetID)
	if err != nil {
		log.Printf("Failed to get balance from Fireblocks: %v", err)

		if statusCode >= 400 && statusCode < 500 {
			http.Error(w, "Invalid request", http.StatusBadRequest)
		} else {
			http.Error(w, "Service unavailable", http.StatusInternalServerError)
		}
		return
	}

	response := GetWalletBalanceResponse{
		ID:           fbResp.ID,
		Total:        fbResp.Total,
		Balance:      fbResp.Balance,
		Available:    fbResp.Available,
		Pending:      fbResp.Pending,
		Frozen:       fbResp.Frozen,
		LockedAmount: fbResp.LockedAmount,
		Staked:       fbResp.Staked,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (h *WalletHandler) GetDepositAddress(w http.ResponseWriter, r *http.Request) {
	walletID := r.PathValue("walletId")
	assetID := r.PathValue("assetId")

	if walletID == "" || assetID == "" {
		http.Error(w, "Wallet ID and Asset ID are required", http.StatusBadRequest)
		return
	}

	wallet, err := h.walletRepo.GetByID(walletID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Wallet not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to get wallet: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	fbResp, statusCode, err := h.fireblocksClient.GetVaultAccountAssetAddresses(wallet.VaultAccountID, assetID)
	if err != nil {
		log.Printf("Failed to get addresses from Fireblocks: %v", err)

		if statusCode >= 400 && statusCode < 500 {
			http.Error(w, "Invalid request", http.StatusBadRequest)
		} else {
			http.Error(w, "Service unavailable", http.StatusInternalServerError)
		}
		return
	}

	if len(fbResp.Addresses) == 0 {
		log.Printf("No addresses found for vault %s, asset %s", wallet.VaultAccountID, assetID)
		http.Error(w, "No deposit address available", http.StatusNotFound)
		return
	}

	firstAddress := fbResp.Addresses[0]

	response := GetDepositAddressResponse{
		AssetID:       firstAddress.AssetID,
		Address:       firstAddress.Address,
		AddressFormat: firstAddress.AddressFormat,
		Type:          firstAddress.Type,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (h *WalletHandler) InitiateTransfer(w http.ResponseWriter, r *http.Request) {
	walletID := r.PathValue("walletId")

	if walletID == "" {
		http.Error(w, "Wallet ID is required", http.StatusBadRequest)
		return
	}

	var req InitiateTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AssetID == "" {
		http.Error(w, "Asset ID is required", http.StatusBadRequest)
		return
	}
	if req.Amount == "" {
		http.Error(w, "Amount is required", http.StatusBadRequest)
		return
	}
	if req.DestinationAddress == "" {
		http.Error(w, "Destination address is required", http.StatusBadRequest)
		return
	}

	wallet, err := h.walletRepo.GetByID(walletID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Wallet not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to get wallet: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Validating balance for wallet %s, asset %s", walletID, req.AssetID)
	balanceResp, statusCode, err := h.fireblocksClient.GetVaultAccountAssetBalance(wallet.VaultAccountID, req.AssetID)
	if err != nil {
		log.Printf("Failed to get balance for validation: %v", err)

		if statusCode >= 400 && statusCode < 500 {
			http.Error(w, "Invalid asset or wallet", http.StatusBadRequest)
		} else {
			http.Error(w, "Unable to validate balance", http.StatusInternalServerError)
		}
		return
	}

	availableBalance, err := strconv.ParseFloat(balanceResp.Available, 64)
	if err != nil {
		log.Printf("Invalid balance format from Fireblocks: %s", balanceResp.Available)
		http.Error(w, "Unable to validate balance", http.StatusInternalServerError)
		return
	}

	transferAmount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		http.Error(w, "Invalid amount format", http.StatusBadRequest)
		return
	}

	if transferAmount > availableBalance {
		log.Printf("Insufficient balance: requested %f, available %f", transferAmount, availableBalance)
		http.Error(w, "Insufficient balance", http.StatusBadRequest)
		return
	}

	fbReq := fireblocks.NewVaultTransferRequest(
		req.AssetID,
		wallet.VaultAccountID,
		req.DestinationAddress,
		req.Amount,
		req.Note,
	)

	fbResp, statusCode, err := h.fireblocksClient.CreateTransaction(fbReq)
	if err != nil {
		log.Printf("Failed to create transaction in Fireblocks: %v", err)

		if statusCode >= 400 && statusCode < 500 {
			http.Error(w, "Invalid request", http.StatusBadRequest)
		} else {
			http.Error(w, "Service unavailable", http.StatusInternalServerError)
		}
		return
	}

	response := InitiateTransferResponse{
		TransactionID:      fbResp.ID,
		Status:             fbResp.Status,
		AssetID:            req.AssetID,
		Amount:             req.Amount,
		DestinationAddress: req.DestinationAddress,
		Note:               req.Note,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err = json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
