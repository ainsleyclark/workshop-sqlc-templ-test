package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"workshop/cmd/themes/internal/site"
	"workshop/internal/common/logs"
	"workshop/internal/common/server"
	"workshop/internal/domain/sighting/stores/sightingsqlite"
	"workshop/internal/infra/config"
	"workshop/internal/infra/db/sqlite"
	"workshop/migrations"

	"github.com/urfave/cli/v3"

	db "workshop/internal/infra/db/sqlc"
)

func main() {
	logger := logs.NewLocal(os.Stderr, slog.LevelInfo)

	// The same configuration as the starter, so DB_URI and .env point
	// both programs at one database.
	cfg, err := config.Load()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	cmd := &cli.Command{
		Name:  "themes",
		Usage: "Serve the themed sighting pages",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "port to listen on",
				Value:   8083,
			},
			&cli.StringFlag{
				Name:  "db",
				Usage: "SQLite database URL",
				Value: cfg.DatabaseURL,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return serve(ctx, logger, cmd.String("db"), cmd.Int("port"))
		},
	}

	if err = cmd.Run(context.Background(), os.Args); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

// serve reads sightings from the database and serves the themes until
// interrupted.
func serve(ctx context.Context, logger *slog.Logger, dbURL string, port int) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, err := sqlite.New(ctx, dbURL)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck

	// A half-migrated database would fail on the first page load, so
	// refuse to start instead, as the starter's web command does.
	migrate, err := migrations.New(conn)
	if err != nil {
		return err
	}
	if pending, err := migrate.HasPending(ctx); err != nil {
		return fmt.Errorf("checking database migrations: %w", err)
	} else if pending {
		return errors.New("database migrations are pending, run 'workshop db migrate up' first")
	}

	mux := http.NewServeMux()
	site.New(logger, sightingsqlite.New(db.New(conn))).Routes(mux)

	logger.Info("Listening", "url", fmt.Sprintf("http://localhost:%d", port))

	return server.New(port, logs.RequestLogger(logger)(mux)).Start(ctx)
}
