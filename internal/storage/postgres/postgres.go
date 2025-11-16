package postgres

import (
	"context"
	"embed"
	"fmt"
	"sort"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type DB struct {
	conn *gorm.DB
}

func New(ctx context.Context, dsn string) (*DB, error) {
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("init gorm postgres: %w", err)
	}

	if err := runMigrations(ctx, gormDB); err != nil {
		sqlDB, dbErr := gormDB.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}

	return &DB{conn: gormDB}, nil
}

func (db *DB) Conn() *gorm.DB {
	return db.conn
}

func (db *DB) Close() {
	sqlDB, err := db.conn.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func runMigrations(ctx context.Context, gormDB *gorm.DB) error {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		sqlBytes, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		if err := gormDB.WithContext(ctx).Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}
