package pddb

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/steinarvk/playdough/pkg/logging"
	"github.com/steinarvk/playdough/pkg/pderr"
	"go.uber.org/zap"
)

//go:embed sql/migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(ctx context.Context, db *sql.DB) error {
	logger := logging.FromContext(ctx)

	sourceDriver, err := iofs.New(migrationsFS, "sql/migrations")
	if err != nil {
		return pderr.Wrap("unable to run migrations", err)
	}

	dbInstance, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return pderr.Wrap("unable to run migrations", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbInstance)
	if err != nil {
		return pderr.Wrap("unable to run migrations", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return pderr.Wrap("unable to check current migration status", err)
	}

	logger.Info("checked current migration status", zap.Uint("current_version", version), zap.Bool("dirty", dirty))

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			logger.Debug("no migrations to run")
			return nil
		}
		return fmt.Errorf("unable to run migrations: %w", err)
	} else {
		logger.Info("successfully applied migrations")
		zap.L().Sugar().Infof("Migrations run successfully")

		version, dirty, err := m.Version()
		if err != nil {
			return fmt.Errorf("error checking migrations after running: %w", err)
		}

		logger.Info("new migration status", zap.Uint("current_version", version), zap.Bool("dirty", dirty))
	}

	return nil
}
