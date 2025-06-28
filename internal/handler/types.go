package handler

type CreateWalletRequest struct {
	Name string `json:"name"`
}

type CreateWalletResponse struct {
	ID                       string `json:"id"`
	Name                     string `json:"name"`
	FireblocksVaultAccountID string `json:"fireblocksVaultAccountID"`
}
