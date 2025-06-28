package fireblocks

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	privateKey *rsa.PrivateKey
	httpClient *http.Client
}

func NewClient(baseURL string, apiKey string, privateKey *rsa.PrivateKey) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		privateKey: privateKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) CreateVaultAccount(req CreateVaultAccountRequest) (*CreateVaultAccountResponse, int, error) {
	respBytes, statusCode, err := c.makeAPIRequest("POST", "/v1/vault/accounts", req)
	if err != nil {
		return nil, 0, err
	}

	if statusCode == http.StatusOK {
		var response CreateVaultAccountResponse
		if err = json.Unmarshal(respBytes, &response); err != nil {
			return nil, statusCode, fmt.Errorf("failed to parse response: %w", err)
		}
		return &response, statusCode, nil
	}

	var fbError ErrorResponse
	if err = json.Unmarshal(respBytes, &fbError); err == nil {
		return nil, statusCode, fbError
	}

	// fallback for unexpected error format
	return nil, statusCode, fmt.Errorf("unexpected API response: %s", string(respBytes))
}

func (c *Client) GetVaultAccountAssetBalance(vaultAccountID, assetID string) (*GetVaultAccountAssetBalanceResponse, int, error) {
	path := fmt.Sprintf("/v1/vault/accounts/%s/%s", vaultAccountID, assetID)

	respBytes, statusCode, err := c.makeAPIRequest("GET", path, nil)
	if err != nil {
		return nil, 0, err
	}

	if statusCode == http.StatusOK {
		var response GetVaultAccountAssetBalanceResponse
		if err = json.Unmarshal(respBytes, &response); err != nil {
			return nil, statusCode, fmt.Errorf("failed to parse response: %w", err)
		}
		return &response, statusCode, nil
	}

	var fbError ErrorResponse
	if err = json.Unmarshal(respBytes, &fbError); err == nil {
		return nil, statusCode, fbError
	}

	// fallback for unexpected error format
	return nil, statusCode, fmt.Errorf("unexpected API response: %s", string(respBytes))
}

// GetAccountsPaged is used for testing only
func (c *Client) GetAccountsPaged() ([]byte, error) {
	path := "/v1/vault/accounts_paged"
	resp, _, err := c.makeAPIRequest("GET", path, nil)
	return resp, err
}

func (c *Client) makeAPIRequest(method, path string, body interface{}) ([]byte, int, error) {
	url := c.baseURL + path

	var reqBodyBytes []byte
	if body != nil {
		var err error
		reqBodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	token, err := c.signJWT(path, reqBodyBytes)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to sign JWT: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("X-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBodyBytes, resp.StatusCode, nil
}

func (c *Client) signJWT(uri string, bodyBytes []byte) (string, error) {
	nonce := uuid.New().String()
	now := time.Now().Unix()
	exp := now + 30

	h := sha256.New()
	h.Write(bodyBytes)
	bodyHash := hex.EncodeToString(h.Sum(nil))

	claims := jwt.MapClaims{
		"uri":      uri,
		"nonce":    nonce,
		"iat":      now,
		"exp":      exp,
		"sub":      c.apiKey,
		"bodyHash": bodyHash,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
