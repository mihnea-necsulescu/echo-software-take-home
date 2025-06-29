package fireblocks

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateVaultAccount(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func() *httptest.Server
		assert    func(t *testing.T, resp *CreateVaultAccountResponse, statusCode int, err error)
	}{
		{
			name: "success",
			mockSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "/v1/vault/accounts", r.URL.Path)

					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(CreateVaultAccountResponse{ID: "123", Name: "Test"})
				}))
			},
			assert: func(t *testing.T, resp *CreateVaultAccountResponse, statusCode int, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, statusCode)
				assert.NotNil(t, resp)
				assert.Equal(t, "123", resp.ID)
				assert.Equal(t, "Test", resp.Name)
			},
		},
		{
			name: "fireblocks_error_unauthorized",
			mockSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(ErrorResponse{Code: -3, Message: "Unauthorized"})
				}))
			},
			assert: func(t *testing.T, resp *CreateVaultAccountResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusUnauthorized, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "Unauthorized")

				var fbErr ErrorResponse
				assert.True(t, errors.As(err, &fbErr))
				assert.Equal(t, -3, fbErr.Code)
			},
		},
		{
			name: "network_error",
			mockSetup: func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				server.Close() // close right away to cause connection error
				return server
			},
			assert: func(t *testing.T, resp *CreateVaultAccountResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "connection refused")
			},
		},
		{
			name: "unexpected_error_format",
			mockSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("<html><body>Internal Server Error</body></html>"))
				}))
			},
			assert: func(t *testing.T, resp *CreateVaultAccountResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusInternalServerError, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "unexpected API response")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.mockSetup()
			defer server.Close()

			testPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			assert.NoError(t, err)

			client := NewClient(server.URL, "test-api-key", testPrivateKey)
			resp, statusCode, err := client.CreateVaultAccount(CreateVaultAccountRequest{Name: "Test"})

			tt.assert(t, resp, statusCode, err)
		})
	}
}

func TestGetVaultAccountBalanceAsset(t *testing.T) {
	tests := []struct {
		name           string
		vaultAccountID string
		assetID        string
		mockSetup      func(vaultAccountID, assetID string) *httptest.Server
		assert         func(t *testing.T, resp *GetVaultAccountAssetBalanceResponse, statusCode int, err error)
	}{
		{
			name:           "success",
			vaultAccountID: "123",
			assetID:        "BTC_TEST",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, fmt.Sprintf("/v1/vault/accounts/%s/%s", vaultAccountID, assetID), r.URL.Path)
					assert.NotEmpty(t, r.Header.Get("X-API-Key"))
					assert.NotEmpty(t, r.Header.Get("Authorization"))

					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(GetVaultAccountAssetBalanceResponse{
						ID:           "BTC_TEST",
						Total:        "0.0003368",
						Balance:      "0.0003368",
						Available:    "0.0003368",
						Pending:      "0",
						Frozen:       "0",
						LockedAmount: "0",
						Staked:       "0",
						BlockHeight:  "4443168",
					})
				}))
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetBalanceResponse, statusCode int, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, statusCode)
				assert.NotNil(t, resp)
				assert.Equal(t, "BTC_TEST", resp.ID)
				assert.Equal(t, "0.0003368", resp.Total)
				assert.Equal(t, "0.0003368", resp.Available)
				assert.Equal(t, "0", resp.Pending)
				assert.Equal(t, "0", resp.Frozen)
				assert.Equal(t, "0", resp.LockedAmount)
				assert.Equal(t, "0", resp.Staked)
				assert.Equal(t, "4443168", resp.BlockHeight)
			},
		},
		{
			name:           "not_found",
			vaultAccountID: "123",
			assetID:        "INVALID",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, fmt.Sprintf("/v1/vault/accounts/%s/%s", vaultAccountID, assetID), r.URL.Path)

					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(ErrorResponse{Code: 1006, Message: "Not found"})
				}))
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetBalanceResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusNotFound, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "Not found")

				var fbErr ErrorResponse
				assert.True(t, errors.As(err, &fbErr))
				assert.Equal(t, 1006, fbErr.Code)
				assert.Equal(t, "Not found", fbErr.Message)
			},
		},
		{
			name:           "invalid_account_id",
			vaultAccountID: "invalid_account_id",
			assetID:        "BTC_TEST",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, fmt.Sprintf("/v1/vault/accounts/%s/%s", vaultAccountID, assetID), r.URL.Path)

					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(ErrorResponse{Code: 11001, Message: "The Provided Vault Account ID is invalid: invalid_account_id"})
				}))
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetBalanceResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusNotFound, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "The Provided Vault Account ID is invalid")

				var fbErr ErrorResponse
				assert.True(t, errors.As(err, &fbErr))
				assert.Equal(t, 11001, fbErr.Code)
				assert.Equal(t, "The Provided Vault Account ID is invalid: invalid_account_id", fbErr.Message)
			},
		},
		{
			name: "fireblocks_error_unauthorized",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(ErrorResponse{Code: -3, Message: "Unauthorized"})
				}))
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetBalanceResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusUnauthorized, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "Unauthorized")

				var fbErr ErrorResponse
				assert.True(t, errors.As(err, &fbErr))
				assert.Equal(t, -3, fbErr.Code)
			},
		},
		{
			name: "network_error",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				server.Close() // close right away to cause connection error
				return server
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetBalanceResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, 0, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "connection refused")
			},
		},
		{
			name: "unexpected_error_format",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("<html><body>Internal Server Error</body></html>"))
				}))
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetBalanceResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusInternalServerError, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "unexpected API response")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.mockSetup(tt.vaultAccountID, tt.assetID)
			defer server.Close()

			testPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			assert.NoError(t, err)

			client := NewClient(server.URL, "test-api-key", testPrivateKey)
			resp, statusCode, err := client.GetVaultAccountAssetBalance(tt.vaultAccountID, tt.assetID)

			tt.assert(t, resp, statusCode, err)
		})
	}
}

func TestGetVaultAccountAssetAddresses(t *testing.T) {
	tests := []struct {
		name           string
		vaultAccountID string
		assetID        string
		mockSetup      func(vaultAccountID, assetID string) *httptest.Server
		assert         func(t *testing.T, resp *GetVaultAccountAssetAddressesResponse, statusCode int, err error)
	}{
		{
			name:           "success",
			vaultAccountID: "123",
			assetID:        "BTC_TEST",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, fmt.Sprintf("/v1/vault/accounts/%s/%s/addresses_paginated", vaultAccountID, assetID), r.URL.Path)
					assert.NotEmpty(t, r.Header.Get("X-API-Key"))
					assert.NotEmpty(t, r.Header.Get("Authorization"))

					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(GetVaultAccountAssetAddressesResponse{
						Addresses: []VaultAccountAddress{
							{
								AssetID:           "BTC_TEST",
								Address:           "tb1q24jg2svw7430u3slcp0rlml7u2tse3h53q0jwe",
								Description:       "",
								Tag:               "",
								Type:              "Permanent",
								AddressFormat:     "SEGWIT",
								LegacyAddress:     "moJU9ea6HdEqMWsyGc892AxiArv2Jyfk7d",
								EnterpriseAddress: "",
								Bip44AddressIndex: 0,
								UserDefined:       false,
							},
							{
								AssetID:           "BTC_TEST",
								Address:           "moJU9ea6HdEqMWsyGc892AxiArv2Jyfk7d",
								Description:       "",
								Tag:               "",
								Type:              "Permanent",
								AddressFormat:     "LEGACY",
								LegacyAddress:     "",
								EnterpriseAddress: "",
								Bip44AddressIndex: 0,
								UserDefined:       false,
							},
						},
					})
				}))
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetAddressesResponse, statusCode int, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, statusCode)
				assert.NotNil(t, resp)
				assert.Len(t, resp.Addresses, 2)

				firstAddr := resp.Addresses[0]
				assert.Equal(t, "BTC_TEST", firstAddr.AssetID)
				assert.Equal(t, "tb1q24jg2svw7430u3slcp0rlml7u2tse3h53q0jwe", firstAddr.Address)
				assert.Equal(t, "SEGWIT", firstAddr.AddressFormat)
				assert.Equal(t, "Permanent", firstAddr.Type)
				assert.False(t, firstAddr.UserDefined)

				secondAddr := resp.Addresses[1]
				assert.Equal(t, "LEGACY", secondAddr.AddressFormat)
				assert.Equal(t, "moJU9ea6HdEqMWsyGc892AxiArv2Jyfk7d", secondAddr.Address)
			},
		},
		{
			name:           "asset_not_found",
			vaultAccountID: "123",
			assetID:        "INVALID_ASSET",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, fmt.Sprintf("/v1/vault/accounts/%s/%s/addresses_paginated", vaultAccountID, assetID), r.URL.Path)

					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(ErrorResponse{Code: 1006, Message: "Not found"})
				}))
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetAddressesResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusNotFound, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "Not found")

				var fbErr ErrorResponse
				assert.True(t, errors.As(err, &fbErr))
				assert.Equal(t, 1006, fbErr.Code)
				assert.Equal(t, "Not found", fbErr.Message)
			},
		},
		{
			name:           "invalid_vault_account",
			vaultAccountID: "invalid-vault",
			assetID:        "BTC_TEST",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, fmt.Sprintf("/v1/vault/accounts/%s/%s/addresses_paginated", vaultAccountID, assetID), r.URL.Path)

					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(ErrorResponse{Code: 11001, Message: "The Provided Vault Account ID is invalid: invalid-vault"})
				}))
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetAddressesResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusBadRequest, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "The Provided Vault Account ID is invalid")

				var fbErr ErrorResponse
				assert.True(t, errors.As(err, &fbErr))
				assert.Equal(t, 11001, fbErr.Code)
			},
		},
		{
			name: "fireblocks_error_unauthorized",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(ErrorResponse{Code: -3, Message: "Unauthorized"})
				}))
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetAddressesResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusUnauthorized, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "Unauthorized")

				var fbErr ErrorResponse
				assert.True(t, errors.As(err, &fbErr))
				assert.Equal(t, -3, fbErr.Code)
			},
		},
		{
			name: "network_error",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				server.Close()
				return server
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetAddressesResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, 0, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "connection refused")
			},
		},
		{
			name: "unexpected_error_format",
			mockSetup: func(vaultAccountID, assetID string) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("<html><body>Internal Server Error</body></html>"))
				}))
			},
			assert: func(t *testing.T, resp *GetVaultAccountAssetAddressesResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusInternalServerError, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "unexpected API response")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.mockSetup(tt.vaultAccountID, tt.assetID)
			defer server.Close()

			testPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			assert.NoError(t, err)

			client := NewClient(server.URL, "test-api-key", testPrivateKey)
			resp, statusCode, err := client.GetVaultAccountAssetAddresses(tt.vaultAccountID, tt.assetID)

			tt.assert(t, resp, statusCode, err)
		})
	}
}

func TestCreateTransaction(t *testing.T) {
	tests := []struct {
		name      string
		request   CreateTransactionRequest
		mockSetup func() *httptest.Server
		assert    func(t *testing.T, resp *CreateTransactionResponse, statusCode int, err error)
	}{
		{
			name: "success",
			request: CreateTransactionRequest{
				Operation: "TRANSFER",
				AssetID:   "BTC_TEST",
				Source: TransactionSource{
					Type: "VAULT_ACCOUNT",
					ID:   "123",
				},
				Destination: TransactionDestination{
					Type: "ONE_TIME_ADDRESS",
					OneTimeAddress: OneTimeAddress{
						Address: "tb1qerzrlxcfu24davlur5sqmgzzgsal6wusda40er",
					},
				},
				Amount: "0.0000001",
				Note:   "Test transfer",
			},
			mockSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "/v1/transactions", r.URL.Path)
					assert.NotEmpty(t, r.Header.Get("X-API-Key"))
					assert.NotEmpty(t, r.Header.Get("Authorization"))
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

					var receivedReq CreateTransactionRequest
					err := json.NewDecoder(r.Body).Decode(&receivedReq)
					assert.NoError(t, err)
					assert.Equal(t, "TRANSFER", receivedReq.Operation)
					assert.Equal(t, "BTC_TEST", receivedReq.AssetID)
					assert.Equal(t, "123", receivedReq.Source.ID)
					assert.Equal(t, "tb1qerzrlxcfu24davlur5sqmgzzgsal6wusda40er", receivedReq.Destination.OneTimeAddress.Address)

					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(CreateTransactionResponse{
						ID:     "eff51bfd-8cec-4b77-b01e-b1aff84dcf49",
						Status: "PENDING_AML_SCREENING",
					})
				}))
			},
			assert: func(t *testing.T, resp *CreateTransactionResponse, statusCode int, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, statusCode)
				assert.NotNil(t, resp)
				assert.Equal(t, "eff51bfd-8cec-4b77-b01e-b1aff84dcf49", resp.ID)
				assert.Equal(t, "PENDING_AML_SCREENING", resp.Status)
			},
		},
		{
			name: "invalid_vault_account",
			request: CreateTransactionRequest{
				Operation: "TRANSFER",
				AssetID:   "BTC_TEST",
				Source: TransactionSource{
					Type: "VAULT_ACCOUNT",
					ID:   "invalid-vault",
				},
				Destination: TransactionDestination{
					Type: "ONE_TIME_ADDRESS",
					OneTimeAddress: OneTimeAddress{
						Address: "tb1qerzrlxcfu24davlur5sqmgzzgsal6wusda40er",
					},
				},
				Amount: "0.0000001",
			},
			mockSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(ErrorResponse{Code: 11001, Message: "The Provided Vault Account ID is invalid: invalid-vault"})
				}))
			},
			assert: func(t *testing.T, resp *CreateTransactionResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusBadRequest, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "The Provided Vault Account ID is invalid")

				var fbErr ErrorResponse
				assert.True(t, errors.As(err, &fbErr))
				assert.Equal(t, 11001, fbErr.Code)
			},
		},
		{
			name: "fireblocks_unauthorized",
			request: CreateTransactionRequest{
				Operation: "TRANSFER",
				AssetID:   "BTC_TEST",
				Source: TransactionSource{
					Type: "VAULT_ACCOUNT",
					ID:   "123",
				},
				Destination: TransactionDestination{
					Type: "ONE_TIME_ADDRESS",
					OneTimeAddress: OneTimeAddress{
						Address: "tb1qerzrlxcfu24davlur5sqmgzzgsal6wusda40er",
					},
				},
				Amount: "0.0000001",
			},
			mockSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(ErrorResponse{Code: -3, Message: "Unauthorized"})
				}))
			},
			assert: func(t *testing.T, resp *CreateTransactionResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusUnauthorized, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "Unauthorized")

				var fbErr ErrorResponse
				assert.True(t, errors.As(err, &fbErr))
				assert.Equal(t, -3, fbErr.Code)
			},
		},
		{
			name: "network_error",
			request: CreateTransactionRequest{
				Operation: "TRANSFER",
				AssetID:   "BTC_TEST",
				Source: TransactionSource{
					Type: "VAULT_ACCOUNT",
					ID:   "123",
				},
				Destination: TransactionDestination{
					Type: "ONE_TIME_ADDRESS",
					OneTimeAddress: OneTimeAddress{
						Address: "tb1qerzrlxcfu24davlur5sqmgzzgsal6wusda40er",
					},
				},
				Amount: "0.0000001",
			},
			mockSetup: func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				server.Close()
				return server
			},
			assert: func(t *testing.T, resp *CreateTransactionResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, 0, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "connection refused")
			},
		},
		{
			name: "unexpected_error_format",
			request: CreateTransactionRequest{
				Operation: "TRANSFER",
				AssetID:   "BTC_TEST",
				Source: TransactionSource{
					Type: "VAULT_ACCOUNT",
					ID:   "123",
				},
				Destination: TransactionDestination{
					Type: "ONE_TIME_ADDRESS",
					OneTimeAddress: OneTimeAddress{
						Address: "tb1qerzrlxcfu24davlur5sqmgzzgsal6wusda40er",
					},
				},
				Amount: "0.0000001",
			},
			mockSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("<html><body>Internal Server Error</body></html>"))
				}))
			},
			assert: func(t *testing.T, resp *CreateTransactionResponse, statusCode int, err error) {
				assert.Error(t, err)
				assert.Equal(t, http.StatusInternalServerError, statusCode)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "unexpected API response")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.mockSetup()
			defer server.Close()

			testPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			assert.NoError(t, err)

			client := NewClient(server.URL, "test-api-key", testPrivateKey)
			resp, statusCode, err := client.CreateTransaction(tt.request)

			tt.assert(t, resp, statusCode, err)
		})
	}
}
