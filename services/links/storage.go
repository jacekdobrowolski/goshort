package links

import (
	"context"
	"database/sql"
	"embed"
	"log/slog"

	"github.com/jmoiron/sqlx"
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
		return nil, err
	}
	logger.Info("Connected to postgres store")

	goose.SetLogger(slog.NewLogLogger(logger.Handler(), slog.LevelInfo))
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	if err := goose.UpToContext(ctx, sqlDB, "migrations", migrationVersion); err != nil {
		panic(err)
	}

	if err := goose.DownToContext(ctx, sqlDB, "migrations", migrationVersion); err != nil {
		panic(err)
	}

	db := sqlx.NewDb(sqlDB, "postgres")

	return &PostgresStore{
		db: db,
	}, nil
}

func (pg *PostgresStore) addLink(short, url string) error {
	_, err := pg.db.Query("INSERT INTO links (short, original) VALUES ($1, $2)", short, url)
	return err
}

func (pg *PostgresStore) getOriginal(short string) (*string, error) {
	rows, err := pg.db.Query("SELECT original FROM links WHERE short = $1", short)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var original string
	for rows.Next() {
		if err := rows.Scan(&original); err != nil {
			return nil, err
		}
	}
	return &original, nil
}
