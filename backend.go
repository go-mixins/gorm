package gorm

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	gormigrate "github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Backend implements generic database backend
type Backend struct {
	DB              *gorm.DB
	Driver          gorm.Dialector
	Config          *gorm.Config
	Debug           bool
	Migrate         bool
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	InitSchema      func(*gorm.DB) error
	Logger          *slog.Logger

	loglevel logger.LogLevel
}

func (b *Backend) config() *gorm.Config {
	if b.Config != nil {
		return b.Config
	}
	config := &gorm.Config{Logger: b, TranslateError: true}
	return config
}

func (b *Backend) WithDebug() *Backend {
	res := *b
	res.DB = res.DB.Debug()
	return &res
}

// WithContext creates Backend clone with new context and logger
func (b *Backend) WithContext(ctx context.Context) *Backend {
	res := new(Backend)
	*res = *b
	res.DB = b.DB.WithContext(ctx)
	return res
}

// Context returns context associated with Backend
func (b *Backend) Context() context.Context {
	if b.DB.Statement == nil {
		return nil
	}
	return b.DB.Statement.Context
}

// Connect sets up the backend and applies migrations if Migrate flag is set to
// true. InitSchema func if set, is used to create initial schema.
func (b *Backend) Connect(migrations ...*gormigrate.Migration) error {
	db, err := gorm.Open(b.Driver, b.config())
	if err != nil {
		return fmt.Errorf("create database connection: %w", err)
	}
	if b.Debug {
		db = db.Debug()
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to acquire DB object: %w", err)
	}
	if b.MaxOpenConns != 0 {
		sqlDB.SetMaxOpenConns(b.MaxOpenConns)
	}
	if b.MaxIdleConns != 0 {
		sqlDB.SetMaxIdleConns(b.MaxIdleConns)
	}
	if b.ConnMaxLifetime != 0 {
		sqlDB.SetConnMaxLifetime(b.ConnMaxLifetime)
	}
	b.DB = db
	if !b.Migrate {
		return nil
	}
	m := gormigrate.New(db, &gormigrate.Options{
		UseTransaction: b.Driver.Name() == "postgres",
	}, migrations)
	if b.InitSchema != nil {
		m.InitSchema(b.InitSchema)
	}
	if err := m.Migrate(); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// Close DB connection
func (b *Backend) Close() error {
	return nil
}
