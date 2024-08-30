package pdtestutils

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steinarvk/playdough/pkg/ezcobra"
	"github.com/steinarvk/playdough/pkg/pddb"
)

func MakeTestingCommandGroup() *cobra.Command {
	group := &cobra.Command{
		Use:   "testing",
		Short: "utilities for testing playdough",
	}

	postgresGroup := &cobra.Command{
		Use:   "docker-postgres",
		Short: "utilities for managing a dockerized postgres instance for testing",
	}

	var postgresParams TestingPostgresOptions

	postgresGroup.PersistentFlags().StringVar(&postgresParams.UnixSocketPath, "unix-socket-path", "/tmp/playdough-testing-postgres/pgsocket/", "path to the unix socket for the testing postgres instance")
	postgresGroup.PersistentFlags().StringVar(&postgresParams.DataDirectory, "data-dir", "", "data directory path (blank for tmpfs)")
	postgresGroup.PersistentFlags().StringVar(&postgresParams.PostgresImage, "postgres-image", "postgres:16.4", "postgres image to use")
	postgresParams.RemoveExistingContainer = true

	postgresGroup.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "run dockerized postgres instance for testing",
		Run: ezcobra.RunNoArgs(func(ctx context.Context) error {
			return RunTestingPostgresContainer(ctx, postgresParams, func(ctx context.Context, conn DatabaseConnectionInfo) error {
				connectCommandline := fmt.Sprintf("psql -h %s/ -U %q -d %q", conn.UnixSocketPath, conn.Username, conn.DatabaseName)
				fmt.Printf("command line to connect:\n\t%s\n", connectCommandline)
				return nil
			})
		}),
	})

	postgresGroup.AddCommand(&cobra.Command{
		Use:   "run-and-migrate",
		Short: "run dockerized postgres instance for testing, and migrate it",
		Run: ezcobra.RunNoArgs(func(ctx context.Context) error {
			return RunTestingPostgresContainer(ctx, postgresParams, func(ctx context.Context, conn DatabaseConnectionInfo) error {
				connectionString := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable", conn.UnixSocketPath, conn.Username, conn.DatabaseName)

				db, err := sql.Open("postgres", connectionString)
				if err != nil {
					return err
				}

				if err := pddb.RunMigrations(ctx, db); err != nil {
					return err
				}

				return nil
			})
		}),
	})

	group.AddCommand(postgresGroup)

	return group
}
