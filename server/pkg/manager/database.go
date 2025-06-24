package manager

import (
	"Server/pkg/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"log/slog"
)

var DB *gorm.DB

func Init() {
	dsn := "host=localhost user=admin password=123456 dbname=tasks port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		slog.Error("failed to connect to database: %v", err)
		panic("failed to connect to database")
	}

	log.Println("database connection established.")
	log.Println("running database migrations...")

	err = db.AutoMigrate(&model.Task{})
	if err != nil {
		slog.Error("failed to migrate database: %v", err)
		panic("failed to migrate database")
	}

	err = db.AutoMigrate(&model.Worker{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	DB = db
}
