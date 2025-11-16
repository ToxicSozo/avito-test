package repository

import (
	"context"

	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) WithContext(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx)
}

func (s *Store) DB() *gorm.DB {
	return s.db
}
