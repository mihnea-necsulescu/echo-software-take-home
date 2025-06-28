package handler

import (
	"bytes"
	"encoding/json"
	"firego-wallet-service/internal/fireblocks"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
		mockSetup func() FireblocksClient
		request   CreateWalletRequest
		assert    func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "success",
			mockSetup: func() FireblocksClient {
				return &MockFireblocksClient{
					Response: &fireblocks.CreateVaultAccountResponse{
						ID:   "123",
						Name: "Test",
					},
					StatusCode: http.StatusCreated,
					Error:      nil,
				}
			},
			request: CreateWalletRequest{Name: "Test"},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, recorder.Code)
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

				var response CreateWalletResponse
				err := json.NewDecoder(recorder.Body).Decode(&response)
				assert.NoError(t, err)

				assert.Equal(t, "todo", response.ID) // todo
				assert.Equal(t, "Test", response.Name)
				assert.Equal(t, "123", response.FireblocksVaultAccountID)
			},
		},
		{
			name: "fireblocks_error_unauthorized",
			mockSetup: func() FireblocksClient {
				return &MockFireblocksClient{
					Response:   nil,
					StatusCode: http.StatusUnauthorized,
					Error:      fireblocks.ErrorResponse{Code: -3, Message: "Unauthorized"},
				}
			},
			request: CreateWalletRequest{Name: "Test"},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Invalid request")
			},
		},
		{
			name: "fireblocks_server_error",
			mockSetup: func() FireblocksClient {
				return &MockFireblocksClient{
					Response:   nil,
					StatusCode: http.StatusInternalServerError,
					Error:      fireblocks.ErrorResponse{Code: 1003, Message: "Create vault account failed"},
				}
			},
			request: CreateWalletRequest{Name: "Test"},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Service unavailable")
			},
		},
		{
			name: "invalid_request_empty_name",
			mockSetup: func() FireblocksClient {
				return &MockFireblocksClient{}
			},
			request: CreateWalletRequest{Name: ""},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Wallet name is required")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.mockSetup()
			handler := NewWalletHandler(mockClient)

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
