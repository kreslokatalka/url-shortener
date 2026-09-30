package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"url-shortener/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

func New(storagePath string) (*Storage, error) {
	const fn = "storage.postgres.new"
	db, err := pgxpool.New(context.Background(), storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}

	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("%s: %w", fn, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveURL(ctx context.Context, urlToSave, alias string) (int64, error) {
	const fn = "storage.postgres.SaveURL"

	var id int64
	err := s.db.QueryRow(
		ctx,
		`INSERT INTO url(alias, url) VALUES($1, $2)
		 RETURNING id`,
		alias,
		urlToSave,
	).Scan(&id)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {

				return 0, storage.ErrAliasExists
			}
		}

		return 0, fmt.Errorf("%s: %w", fn, err)
	}
	return id, nil
}

func (s *Storage) GetURL(ctx context.Context, alias string) (string, error) {
	const fn = "storage.postgres.GetURL"

	var url string
	err := s.db.QueryRow(ctx,
		"SELECT url FROM url WHERE alias = $1", alias).Scan(&url)
	if err != nil {
		log.Printf("Database error: %v", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", storage.ErrURLNotFound
		}
		return "", fmt.Errorf("%s: execute statement %w", fn, err)
	}
	return url, nil
}

func (s *Storage) DeleteURL(ctx context.Context, alias string) (int64, error) {
	const fn = "storage.postgres.DeleteURL"

	result, err := s.db.Exec(ctx,
		"DELETE FROM url WHERE alias = $1", alias)
	if err != nil {
		return 0, fmt.Errorf("%s: execute statement %w", fn, err)
	}

	rowsAffected := result.RowsAffected()

	return rowsAffected, nil
}

func (s *Storage) UpdateURL(ctx context.Context, newURL, NewAlias, alias string) (int64, error) {
	const fn = "storage.postgres.UpdateURL"
	if NewAlias == "" {
		NewAlias = alias
	}

	result, err := s.db.Exec(ctx,
		"UPDATE url SET url = $1, alias = $2 WHERE alias = $3", newURL, NewAlias, alias)

	if err != nil {
		return 0, fmt.Errorf("%s: execute statement %w", fn, err)
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected, nil
}

func (s *Storage) Close() error {
	s.db.Close()
	return nil
}
