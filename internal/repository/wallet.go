package repository

import (
	"firego-wallet-service/internal/model"
	"gorm.io/gorm"
)

type walletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) *walletRepository {
	return &walletRepository{
		db: db,
	}
}

func (r *walletRepository) Create(wallet *model.Wallet) error {
	return r.db.Create(wallet).Error
}
