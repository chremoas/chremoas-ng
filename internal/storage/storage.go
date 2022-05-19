package storage

import sq "github.com/Masterminds/squirrel"

type Storage struct {
	DB *sq.StatementBuilderType
}

func New(db *sq.StatementBuilderType) *Storage {
	return &Storage{DB: db}
}
