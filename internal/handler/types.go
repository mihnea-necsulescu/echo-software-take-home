package handler

type CreateWalletRequest struct {
	Name string `json:"name"`
}

type CreateWalletResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	VaultAccountID string `json:"vaultAccountID"`
}

type GetWalletBalanceResponse struct {
	ID           string `json:"id"`
	Total        string `json:"total"`
	Balance      string `json:"balance"`
	Available    string `json:"available"`
	Pending      string `json:"pending"`
	Frozen       string `json:"frozen"`
	LockedAmount string `json:"lockedAmount"`
	Staked       string `json:"staked"`
}

type GetDepositAddressResponse struct {
	AssetID       string `json:"assetId"`
	Address       string `json:"address"`
	AddressFormat string `json:"addressFormat"`
	Type          string `json:"type"`
}
