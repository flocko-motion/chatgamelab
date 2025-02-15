package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path"
)

var (
	db              *gorm.DB
	fileUsageReport *os.File
	pathUsageReport string
)

func Init() {
	pathDb := path.Join("var", "sqlite.db")
	var err error
	db, err = gorm.Open(sqlite.Open(pathDb), &gorm.Config{})
	if err != nil {
		panic("failed to connect database '" + pathDb + "': " + err.Error())
	}

	pathUsageReport = path.Join("var", "usage_report.csv")
	if fileUsageReport, err = os.OpenFile(pathUsageReport, os.O_CREATE|os.O_WRONLY, 0644); err != nil {
		panic("failed to create or open usage report file: " + err.Error())
	}

	// Migrate the schema
	tables := []interface{}{&User{}, &Game{}, &Session{}, &Chapter{}}
	for _, table := range tables {
		err = db.AutoMigrate(table)
		if err != nil {
			panic("failed to migrate database: " + err.Error())
		}
	}
}
