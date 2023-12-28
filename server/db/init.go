package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path"
)

var db *gorm.DB

func Init() {
	pathDb := path.Join("var", "sqlite.db")
	var err error
	db, err = gorm.Open(sqlite.Open(pathDb), &gorm.Config{})
	if err != nil {
		panic("failed to connect database '" + pathDb + "': " + err.Error())
	}

	// Migrate the schema
	err = db.AutoMigrate(&User{})
	if err != nil {
		panic("failed to migrate database: " + err.Error())
	}
}
