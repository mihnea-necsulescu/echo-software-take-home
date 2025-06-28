package model

import "time"

type Wallet struct {
	ID             string `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name           string `gorm:"not null"`
	VaultAccountID string `gorm:"uniqueIndex;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
