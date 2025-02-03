package links

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	// Postgres driver give me a break.
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

const migrationVersion = 20240305130405

//go:embed migrations/*.sql
var embedMigrations embed.FS

type Store interface {
	addLink(short, url string) error
	getOriginal(short string) (*string, error)
}

type PostgresStore struct {
	db *sqlx.DB
}

func NewPostgresStore(ctx context.Context, connStr string, logger *slog.Logger) (*PostgresStore, error) {
	sqlDB, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to postgres: %w", err)
	}

	logger.Info("connected to postgres store")

	goose.SetLogger(slog.NewLogLogger(logger.Handler(), slog.LevelInfo))
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return nil, fmt.Errorf("error migrating cannot set dialect: %w", err)
	}

	if err := goose.UpToContext(ctx, sqlDB, "migrations", migrationVersion); err != nil {
		return nil, fmt.Errorf("error migration up: %w", err)
	}

	if err := goose.DownToContext(ctx, sqlDB, "migrations", migrationVersion); err != nil {
		return nil, fmt.Errorf("error migrating down: %w", err)
	}

	db := sqlx.NewDb(sqlDB, "postgres")

	return &PostgresStore{
		db: db,
	}, nil
}

func (pg *PostgresStore) addLink(short, url string) error {
	rows, err := pg.db.Query("INSERT INTO links (short, original) VALUES ($1, $2)", short, url)
	if err != nil {
		return fmt.Errorf("error query addLink: %w", err)
	}

	defer rows.Close()

	if rowsErr := rows.Err(); rowsErr != nil {
		return fmt.Errorf("addLink query returned error: %w", rowsErr)
	}

	return nil
}

func (pg *PostgresStore) getOriginal(short string) (*string, error) {
	rows, err := pg.db.Query("SELECT original FROM links WHERE short = $1", short)
	if err != nil {
		return nil, fmt.Errorf("error executing query getOriginal: %w", err)
	}

	defer rows.Close()

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("getOriginal query returned error: %w", rowsErr)
	}

	var original string

	for rows.Next() {
		if err := rows.Scan(&original); err != nil {
			return nil, fmt.Errorf("error scaning rows: %w", err)
		}
	}

	return &original, nil
}
