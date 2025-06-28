package handler

import (
	"bytes"
	"encoding/json"
	"firego-wallet-service/internal/fireblocks"
	"firego-wallet-service/internal/model"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type MockWalletRepository struct {
	CreateError   error
	CreatedWallet *model.Wallet
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

type MockFireblocksClient struct {
	Response   *fireblocks.CreateVaultAccountResponse
	StatusCode int
	Error      error
}

func (m *MockFireblocksClient) CreateVaultAccount(_ fireblocks.CreateVaultAccountRequest) (*fireblocks.CreateVaultAccountResponse, int, error) {
	return m.Response, m.StatusCode, m.Error
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
					Response: &fireblocks.CreateVaultAccountResponse{
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
					Response:   nil,
					StatusCode: http.StatusUnauthorized,
					Error:      fireblocks.ErrorResponse{Code: -3, Message: "Unauthorized"},
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
					Response:   nil,
					StatusCode: http.StatusInternalServerError,
					Error:      fireblocks.ErrorResponse{Code: 1003, Message: "Create vault account failed"},
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
					Response: &fireblocks.CreateVaultAccountResponse{
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
