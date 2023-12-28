package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path"
)

func InitDB() (*gorm.DB, error) {
	pathDb := path.Join("var", "sqlite.db")
	db, err := gorm.Open(sqlite.Open(pathDb), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	err = db.AutoMigrate(&User{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
