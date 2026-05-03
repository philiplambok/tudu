package cmd

import (
	"log"
	"log/slog"

	"github.com/philiplambok/tudu/internal/transport"
	"github.com/philiplambok/tudu/pkg/avatar"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var serveCmd = &cobra.Command{
	RunE:  runServer,
	Use:   "serve",
	Short: "start the HTTP server",
}

func runServer(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig(".")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	slog.SetLogLoggerLevel(cfg.Log.ParseSlogLevel())

	db, err := gorm.Open(postgres.Open(cfg.Database.Source), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	var avatarProvider avatar.Provider
	if cfg.Env == "production" {
		avatarProvider = avatar.NewGravatar()
	} else {
		avatarProvider = avatar.NewMock()
	}

	return transport.NewServer(cfg, db, avatarProvider).Start()
}
