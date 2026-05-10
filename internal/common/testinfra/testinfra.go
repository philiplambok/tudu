package testinfra

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"time"

	testfixtures "github.com/go-testfixtures/testfixtures/v3"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	tududb "github.com/philiplambok/tudu/db"
)

//go:embed fixtures
var fixturesFS embed.FS

func SetupTestDB(ctx context.Context) (*gorm.DB, *postgrescontainer.PostgresContainer, error) {
	container, err := postgrescontainer.Run(ctx, "postgres:16-alpine",
		postgrescontainer.WithDatabase("tudu_test"),
		postgrescontainer.WithUsername("postgres"),
		postgrescontainer.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("start postgres container: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("connection string: %w", err)
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open sql db: %w", err)
	}

	goose.SetBaseFS(tududb.Migrations)
	goose.SetTableName("schema_migrations")
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	subFS, err := fs.Sub(fixturesFS, "fixtures")
	if err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("sub fixtures fs: %w", err)
	}
	loader, err := testfixtures.New(
		testfixtures.Database(sqlDB),
		testfixtures.Dialect("postgresql"),
		testfixtures.FS(subFS),
	)
	if err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("setup fixtures: %w", err)
	}
	if err := loader.Load(); err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("load fixtures: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return nil, nil, fmt.Errorf("close migration db: %w", err)
	}

	if err := container.Snapshot(ctx); err != nil {
		return nil, nil, fmt.Errorf("snapshot: %w", err)
	}

	gormDB, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("gorm open: %w", err)
	}

	return gormDB, container, nil
}

func RestoreDB(ctx context.Context, container *postgrescontainer.PostgresContainer, db **gorm.DB) error {
	if err := container.Restore(ctx); err != nil {
		return fmt.Errorf("restore snapshot: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("connection string: %w", err)
	}

	newDB, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("gorm reconnect: %w", err)
	}
	*db = newDB
	return nil
}
