package database

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
}

func NewConnection(databaseURL string, env string) (*DB, error) {
	config := &gorm.Config{}

	// Set log level based on environment
	if env == "development" {
		config.Logger = logger.Default.LogMode(logger.Info)
	} else {
		config.Logger = logger.Default.LogMode(logger.Error)
	}

	db, err := gorm.Open(postgres.Open(databaseURL), config)
	if err != nil {
		return nil, err
	}

	log.Println("Database connected successfully")
	return &DB{db}, nil
}

func (db *DB) Migrate(models ...interface{}) error {
	return db.AutoMigrate(models...)
}