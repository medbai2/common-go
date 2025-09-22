package database

import (
	"fmt"
	"time"

	"go-common/errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config represents database configuration
type Config struct {
	Driver          string
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	SchemaAutoApply bool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// New creates a new GORM database connection
func New(cfg Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)

	// Configure GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabaseError, "failed to connect to database")
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabaseError, "failed to get underlying sql.DB")
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)

	// Set connection lifetime - configuration already contains proper duration
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	// Set connection idle time - configuration already contains proper duration
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabaseError, "failed to ping database")
	}

	return db, nil
}

// HealthCheck performs a simple health check on the database
func HealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabaseError, "failed to get underlying sql.DB")
	}

	var result int
	err = sqlDB.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabaseError, "health check failed")
	}
	return nil
}
