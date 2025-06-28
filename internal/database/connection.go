package database

import (
	"firego-wallet-service/internal/model"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
)

func Connect(host, port, dbname, user, password, sslmode string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		host, port, dbname, user, password, sslmode)

	log.Printf("Connecting to database with DSN: host=%s port=%s dbname=%s user=%s sslmode=%s",
		host, port, dbname, user, sslmode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Successfully connected to database")

	log.Println("Running migrations...")
	if err = db.AutoMigrate(&model.Wallet{}); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Successfully migrated database")

	return db, nil
}
