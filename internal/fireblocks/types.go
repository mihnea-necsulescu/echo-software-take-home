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

type GetVaultAccountAssetBalanceResponse struct {
	ID           string `json:"id"`
	Total        string `json:"total"`
	Balance      string `json:"balance"`
	Available    string `json:"available"`
	Pending      string `json:"pending"`
	Frozen       string `json:"frozen"`
	LockedAmount string `json:"lockedAmount"`
	Staked       string `json:"staked"`
	BlockHeight  string `json:"blockHeight"`
}

type GetVaultAccountAssetAddressesResponse struct {
	Addresses []VaultAccountAddress `json:"addresses"`
}

type VaultAccountAddress struct {
	AssetID           string `json:"assetId"`
	Address           string `json:"address"`
	Description       string `json:"description"`
	Tag               string `json:"tag"`
	Type              string `json:"type"`
	AddressFormat     string `json:"addressFormat"`
	LegacyAddress     string `json:"legacyAddress"`
	EnterpriseAddress string `json:"enterpriseAddress"`
	Bip44AddressIndex int    `json:"bip44AddressIndex"`
	UserDefined       bool   `json:"userDefined"`
}

type CreateTransactionRequest struct {
	Operation   string                 `json:"operation"`
	AssetID     string                 `json:"assetId"`
	Source      TransactionSource      `json:"source"`
	Destination TransactionDestination `json:"destination"`
	Amount      string                 `json:"amount"`
	Note        string                 `json:"note,omitempty"`
}

type TransactionSource struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type TransactionDestination struct {
	Type           string         `json:"type"`
	OneTimeAddress OneTimeAddress `json:"oneTimeAddress"`
}

type OneTimeAddress struct {
	Address string `json:"address"`
}

type CreateTransactionResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
