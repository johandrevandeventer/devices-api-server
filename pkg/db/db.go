package devicesdb

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type BMS_DB struct {
	DB *gorm.DB
}

var BMS_DB_Instance *BMS_DB

func NewDB() (*BMS_DB, error) {
	var err error

	dsn := os.Getenv("DB_URL")

	if dsn == "" {
		return nil, fmt.Errorf("DB_URL environment variable not set")
	}

	DB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// ✅ Properly configure connection pooling
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(1)                   // Limit max open connections (adjust as needed)
	sqlDB.SetMaxIdleConns(5)                   // Keep up to 5 idle connections
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // Close connections after 30 min

	BMS_DB_Instance = &BMS_DB{DB: DB}

	// Perform health check before returning instance
	if err := BMS_DB_Instance.HealthCheck(); err != nil {
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	return BMS_DB_Instance, nil
}

func GetDB() (*BMS_DB, error) {
	if BMS_DB_Instance == nil {
		db, err := NewDB()
		if err != nil {
			return nil, fmt.Errorf("failed to get database instance: %w", err)
		}
		BMS_DB_Instance = db
	}
	return BMS_DB_Instance, nil
}

func (db *BMS_DB) HealthCheck() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Ping the database to check connection
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database is not reachable: %w", err)
	}

	return nil
}

func (db *BMS_DB) TableExists(tableName string) bool {
	// Get the current database name from the connection string
	var dbName string
	err := db.DB.Raw("SELECT DATABASE()").Scan(&dbName).Error
	if err != nil {
		fmt.Printf("Failed to get database name: %v\n", err)
		return false
	}

	var count int64
	db.DB.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?", dbName, tableName).Count(&count)
	return count > 0
}

func (db *BMS_DB) Migrate(tableName string, target any) error {
	if err := db.DB.AutoMigrate(target); err != nil {
		return fmt.Errorf("failed to migrate table %s: %w", tableName, err)
	}
	return nil
}

func (db *BMS_DB) Close() {
	sqlDB, err := db.DB.DB()
	if err == nil {
		sqlDB.Close()
	}
}
