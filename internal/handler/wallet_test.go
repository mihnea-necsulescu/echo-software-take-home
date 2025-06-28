package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"firego-wallet-service/internal/fireblocks"
	"firego-wallet-service/internal/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type MockWalletRepository struct {
	CreateError   error
	CreatedWallet *model.Wallet

	GetByIDWallet *model.Wallet
	GetByIDError  error
}

func (m *MockWalletRepository) Create(wallet *model.Wallet) error {
	if m.CreateError != nil {
		return m.CreateError
	}

	wallet.ID = "test-wallet-id-123"
	now := time.Now()
	wallet.CreatedAt = now
	wallet.UpdatedAt = now

	m.CreatedWallet = wallet

	return nil
}

func (m *MockWalletRepository) GetByID(_ string) (*model.Wallet, error) {
	if m.GetByIDError != nil {
		return nil, m.GetByIDError
	}

	return m.GetByIDWallet, nil
}

type MockFireblocksClient struct {
	CreateVaultAccountResponse          *fireblocks.CreateVaultAccountResponse
	GetVaultAccountAssetBalanceResponse *fireblocks.GetVaultAccountAssetBalanceResponse

	StatusCode int
	Error      error
}

func (m *MockFireblocksClient) CreateVaultAccount(_ fireblocks.CreateVaultAccountRequest) (*fireblocks.CreateVaultAccountResponse, int, error) {
	return m.CreateVaultAccountResponse, m.StatusCode, m.Error
}

func (m *MockFireblocksClient) GetVaultAccountAssetBalance(_, _ string) (*fireblocks.GetVaultAccountAssetBalanceResponse, int, error) {
	return m.GetVaultAccountAssetBalanceResponse, m.StatusCode, m.Error
}

func (m *MockFireblocksClient) GetVaultAccountAssetAddresses(vaultAccountID, assetID string) (*fireblocks.GetVaultAccountAssetAddressesResponse, int, error) {
	return nil, 0, nil //todo
}

func TestCreateWallet(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func() (WalletRepository, FireblocksClient)
		request   CreateWalletRequest
		assert    func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success",
			mockSetup: func() (WalletRepository, FireblocksClient) {
				mockRepo := &MockWalletRepository{CreateError: nil}
				mockFireblocksClient := &MockFireblocksClient{
					CreateVaultAccountResponse: &fireblocks.CreateVaultAccountResponse{
						ID:   "123",
						Name: "Test",
					},
					StatusCode: http.StatusCreated,
					Error:      nil,
				}
				return mockRepo, mockFireblocksClient
			},
			request: CreateWalletRequest{Name: "Test"},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, recorder.Code)
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

				var response CreateWalletResponse
				err := json.NewDecoder(recorder.Body).Decode(&response)
				assert.NoError(t, err)

				assert.Equal(t, "test-wallet-id-123", response.ID)
				assert.Equal(t, "Test", response.Name)
				assert.Equal(t, "123", response.VaultAccountID)
			},
		},
		{
			name: "fireblocks_error_unauthorized",
			mockSetup: func() (WalletRepository, FireblocksClient) {
				mockFireblocksClient := &MockFireblocksClient{
					CreateVaultAccountResponse: nil,
					StatusCode:                 http.StatusUnauthorized,
					Error:                      fireblocks.ErrorResponse{Code: -3, Message: "Unauthorized"},
				}
				return nil, mockFireblocksClient
			},
			request: CreateWalletRequest{Name: "Test"},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Invalid request")
			},
		},
		{
			name: "fireblocks_server_error",
			mockSetup: func() (WalletRepository, FireblocksClient) {
				mockFireblocksClient := &MockFireblocksClient{
					CreateVaultAccountResponse: nil,
					StatusCode:                 http.StatusInternalServerError,
					Error:                      fireblocks.ErrorResponse{Code: 1003, Message: "Create vault account failed"},
				}
				return nil, mockFireblocksClient
			},
			request: CreateWalletRequest{Name: "Test"},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Service unavailable")
			},
		},
		{
			name: "invalid_request_empty_name",
			mockSetup: func() (WalletRepository, FireblocksClient) {
				return nil, nil
			},
			request: CreateWalletRequest{Name: ""},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Wallet name is required")
			},
		},
		{
			name: "database_error",
			mockSetup: func() (WalletRepository, FireblocksClient) {
				mockRepo := &MockWalletRepository{CreateError: assert.AnError}
				mockFireblocksClient := &MockFireblocksClient{
					CreateVaultAccountResponse: &fireblocks.CreateVaultAccountResponse{
						ID:   "123",
						Name: "Test",
					},
					StatusCode: http.StatusCreated,
					Error:      nil,
				}
				return mockRepo, mockFireblocksClient
			},
			request: CreateWalletRequest{Name: "Test"},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Failed to create wallet")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo, mockClient := tt.mockSetup()
			handler := NewWalletHandler(mockRepo, mockClient)

			reqBody, err := json.Marshal(tt.request)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/wallets", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()

			handler.CreateWallet(recorder, req)

			tt.assert(t, recorder)
		})
	}
}

func TestGetWalletBalance(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func() (WalletRepository, FireblocksClient)
		url       string
		assert    func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success",
			mockSetup: func() (WalletRepository, FireblocksClient) {
				mockRepo := &MockWalletRepository{
					GetByIDWallet: &model.Wallet{
						ID:             "123",
						Name:           "Test",
						VaultAccountID: "vault-account-id",
					},
					GetByIDError: nil,
				}
				mockFireblocksClient := &MockFireblocksClient{
					GetVaultAccountAssetBalanceResponse: &fireblocks.GetVaultAccountAssetBalanceResponse{
						ID:           "BTC_TEST",
						Total:        "0.0003368",
						Balance:      "0.0003368",
						Available:    "0.0003368",
						Pending:      "0",
						Frozen:       "0",
						LockedAmount: "0",
						Staked:       "0",
						BlockHeight:  "4443168",
					},
					StatusCode: http.StatusOK,
					Error:      nil,
				}
				return mockRepo, mockFireblocksClient
			},
			url: "/wallets/123/assets/BTC_TEST/balance",
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

				var response GetWalletBalanceResponse
				err := json.NewDecoder(recorder.Body).Decode(&response)
				assert.NoError(t, err)

				assert.Equal(t, "BTC_TEST", response.ID)
				assert.Equal(t, "0.0003368", response.Total)
				assert.Equal(t, "0.0003368", response.Available)
				assert.Equal(t, "0", response.Pending)
				assert.Equal(t, "0", response.Frozen)
			},
		},
		{
			name: "wallet_not_found",
			mockSetup: func() (WalletRepository, FireblocksClient) {
				mockRepo := &MockWalletRepository{
					GetByIDWallet: nil,
					GetByIDError:  gorm.ErrRecordNotFound,
				}
				return mockRepo, nil
			},
			url: "/wallets/nonexistent-wallet/assets/BTC_TEST/balance",
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Wallet not found")
			},
		},
		{
			name: "database_error",
			mockSetup: func() (WalletRepository, FireblocksClient) {
				mockRepo := &MockWalletRepository{
					GetByIDWallet: nil,
					GetByIDError:  errors.New("database connection failed"),
				}
				return mockRepo, nil
			},
			url: "/wallets/123/assets/BTC_TEST/balance",
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Internal server error")
			},
		},
		{
			name: "fireblocks_asset_not_found",
			mockSetup: func() (WalletRepository, FireblocksClient) {
				mockRepo := &MockWalletRepository{
					GetByIDWallet: &model.Wallet{
						ID:             "123",
						Name:           "Test",
						VaultAccountID: "vault-account-id",
					},
					GetByIDError: nil,
				}
				mockFireblocksClient := &MockFireblocksClient{
					GetVaultAccountAssetBalanceResponse: nil,
					StatusCode:                          http.StatusNotFound,
					Error:                               fireblocks.ErrorResponse{Code: 1006, Message: "Not found"},
				}
				return mockRepo, mockFireblocksClient
			},
			url: "/wallets/123/assets/INVALID_ASSET/balance",
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Invalid request")
			},
		},
		{
			name: "fireblocks_unauthorized",
			mockSetup: func() (WalletRepository, FireblocksClient) {
				mockRepo := &MockWalletRepository{
					GetByIDWallet: &model.Wallet{
						ID:             "123",
						Name:           "Test",
						VaultAccountID: "vault-account-id",
					},
					GetByIDError: nil,
				}
				mockFireblocksClient := &MockFireblocksClient{
					GetVaultAccountAssetBalanceResponse: nil,
					StatusCode:                          http.StatusUnauthorized,
					Error:                               fireblocks.ErrorResponse{Code: -3, Message: "Unauthorized"},
				}
				return mockRepo, mockFireblocksClient
			},
			url: "/wallets/123/assets/BTC_TEST/balance",
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Invalid request")
			},
		},
		{
			name: "fireblocks_server_error",
			mockSetup: func() (WalletRepository, FireblocksClient) {
				mockRepo := &MockWalletRepository{
					GetByIDWallet: &model.Wallet{
						ID:             "123",
						Name:           "Test",
						VaultAccountID: "vault-account-id",
					},
					GetByIDError: nil,
				}
				mockFireblocksClient := &MockFireblocksClient{
					GetVaultAccountAssetBalanceResponse: nil,
					StatusCode:                          http.StatusInternalServerError,
					Error:                               fireblocks.ErrorResponse{Code: 1000, Message: "Internal server error"},
				}
				return mockRepo, mockFireblocksClient
			},
			url: "/wallets/123/assets/BTC_TEST/balance",
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Service unavailable")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo, mockClient := tt.mockSetup()
			handler := NewWalletHandler(mockRepo, mockClient)

			mux := http.NewServeMux()
			mux.HandleFunc("GET /wallets/{walletId}/assets/{assetId}/balance", handler.GetWalletBalance)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			recorder := httptest.NewRecorder()

			mux.ServeHTTP(recorder, req)

			tt.assert(t, recorder)
		})
	}
}
