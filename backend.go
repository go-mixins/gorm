package gorm

import (
	"context"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	gormigrate "gopkg.in/gormigrate.v1"
)

type logLevel int

// Log levels (defaults to Debug)
const (
	LogDebug logLevel = iota
	LogInfo
)

// Backend implements generic database backend
type Backend struct {
	DB              *gorm.DB
	Driver          string
	DBURI           string
	Debug           bool
	LogLevel        logLevel
	Migrate         bool
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	InitSchema      func(*gorm.DB) error

	context context.Context
}

// WithContext creates Backend clone with new context and logger
func (b *Backend) WithContext(ctx context.Context) *Backend {
	res := new(Backend)
	*res = *b
	res.DB = b.DB.New()
	res.context = ctx
	res.DB.SetLogger(newLogger(res.context, res.LogLevel))
	return res
}

// WithLogger injects log printer func into Backend
func (b *Backend) WithLogger(printer func(...interface{})) *Backend {
	res := new(Backend)
	*res = *b
	res.DB = b.DB.New()
	res.DB.SetLogger(logger{printer: printer})
	return res
}

// Context returns context associated with Backend
func (b *Backend) Context() context.Context {
	return b.context
}

func (b *Backend) driver() string {
	if b.Driver != "" {
		return b.Driver
	}
	return "sqlite3"
}

func (b *Backend) dbURI() string {
	driver := b.driver()
	if b.DBURI == "" {
		return "test"
	}
	if driver == "postgres" {
		return b.DBURI + " binary_parameters=yes"
	} else if driver == "mysql" {
		return b.DBURI + "?charset=utf8&parseTime=True&loc=Local"
	}
	return b.DBURI
}

// Connect sets up the backend and applies migrations if Migrate flag is set to
// true. InitSchema func if set, is used to create initial schema.
func (b *Backend) Connect(migrations ...*gormigrate.Migration) error {
	db, err := gorm.Open(b.driver(), b.dbURI())
	if err != nil {
		return fmt.Errorf("create database connection: %w", err)
	}
	if b.MaxOpenConns != 0 {
		db.DB().SetMaxOpenConns(b.MaxOpenConns)
	}
	if b.MaxIdleConns != 0 {
		db.DB().SetMaxIdleConns(b.MaxIdleConns)
	}
	if b.ConnMaxLifetime != 0 {
		db.DB().SetConnMaxLifetime(b.ConnMaxLifetime)
	}
	db.SetLogger(newLogger(b.context, b.LogLevel))
	if b.Debug {
		db.LogMode(true)
	}
	b.DB = db
	if !b.Migrate {
		return nil
	}
	m := gormigrate.New(db, &gormigrate.Options{
		UseTransaction: b.Driver == "postgres",
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
	if err := b.DB.Close(); err != nil {
		return fmt.Errorf("closing database connection: %w", err)
	}
	return nil
}
