package fireblocks

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
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
