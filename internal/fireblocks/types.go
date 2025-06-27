package fireblocks

import "fmt"

type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e ErrorResponse) Error() string {
	return fmt.Sprintf("Fireblocks API error (code %d): %s", e.Code, e.Message)
}

type CreateVaultAccountRequest struct {
	Name string `json:"name"`
}

type CreateVaultAccountResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
