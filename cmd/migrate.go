package cmd

import (
	"context"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

var (
	migrateCmd = &cobra.Command{
		RunE:  runMigration,
		Use:   "migrate",
		Short: "run db migration files under db/migrations",
	}
	migrateRollback bool
	migrateDir      string
)

func init() {
	migrateCmd.Flags().BoolVarP(&migrateRollback, "rollback", "r", false, "rollback the latest migration")
	migrateCmd.PersistentFlags().StringVarP(&migrateDir, "dir", "d", "db/migrations", "migrations directory")
}

func runMigration(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	cfg, err := loadConfig(".")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := goose.OpenDBWithDriver("pgx", cfg.Database.Source)
	if err != nil {
		log.Fatalf("goose: failed to open DB: %v", err)
	}
	defer db.Close()

	goose.SetTableName("schema_migrations")

	if migrateRollback {
		return goose.RunContext(ctx, "down", db, migrateDir)
	}
	return goose.RunContext(ctx, "up", db, migrateDir)
}
