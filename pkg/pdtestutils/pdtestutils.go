package pdtestutils

import (
	"context"
	cryptorand "crypto/rand"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/steinarvk/playdough/pkg/logging"
	"github.com/steinarvk/playdough/pkg/pderr"
	"go.uber.org/zap"
)

/*
mkdir -p data/pgdata
mkdir -p data/pgsocket
docker rm -f /playdough_test_pg
docker run \
    --name playdough_test_pg \
    -e POSTGRES_DB=playdough_test \
    -e POSTGRES_USER=playdough_test \
    -e POSTGRES_PASSWORD=hunter2 \
    --mount type=bind,source="$(pwd)"/data/pgdata,target=/var/lib/postgresql/data \
    --mount type=bind,source="$(pwd)"/data/pgsocket,target=/var/run/postgresql \
    postgres:16.4 \
    -c "unix_socket_directories=/var/run/postgresql" \
    -c "listen_addresses="
*/

type TestingPostgresOptions struct {
	DockerBinary            string
	RemoveExistingContainer bool
	ContainerName           string
	DatabaseName            string
	Username                string
	Password                string
	PostgresImage           string
	UnixSocketPath          string
	DataDirectory           string
}

func generateStrongPassword() (string, error) {
	alphabet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	n := 64

	limit := big.NewInt(int64(len(alphabet)))

	b := make([]byte, n)
	for i := range b {
		bigIndex, err := cryptorand.Int(cryptorand.Reader, limit)
		if err != nil {
			return "", pderr.Wrap("generateStrongPassword: failed to generate random number", err)
		}
		index := int(bigIndex.Int64())
		b[i] = alphabet[index]
	}

	return string(b), nil
}

type DatabaseConnectionInfo struct {
	UnixSocketPath string
	Username       string
	DatabaseName   string
}

func RunTestingPostgresContainer(ctx context.Context, opts TestingPostgresOptions, runWithDatabase func(context.Context, DatabaseConnectionInfo) error) error {
	logger := logging.FromContext(ctx)

	if opts.UnixSocketPath == "" {
		return pderr.MissingRequiredFlag("RunTestingPostgresContainer: UnixSocketPath must be set")
	}

	// Set defaults
	if opts.ContainerName == "" {
		opts.ContainerName = "playdough_test_pg"
	}

	if opts.DatabaseName == "" {
		opts.DatabaseName = "playdough_test"
	}

	if opts.Username == "" {
		opts.Username = "playdough_test"
	}

	if opts.PostgresImage == "" {
		opts.PostgresImage = "postgres:16.4"
	}

	if opts.Password == "" {
		password, err := generateStrongPassword()
		if err != nil {
			return pderr.Wrap("RunTestingPostgresContainer: failed to generate password", err)
		}
		opts.Password = password
	}

	if opts.DockerBinary == "" {
		opts.DockerBinary = "/usr/bin/docker"
	}

	useTmpfs := opts.DataDirectory == ""

	if opts.RemoveExistingContainer {
		rmOldArgv := []string{
			opts.DockerBinary,
			"rm",
			"-f",
			opts.ContainerName,
		}
		logger.Info("removing existing postgres container (if any)", zap.Strings("argv", rmOldArgv))

		if err := exec.CommandContext(ctx, rmOldArgv[0], rmOldArgv[1:]...).Run(); err != nil {
			return pderr.Wrap("RunTestingPostgresContainer: failed to remove existing container", err)
		}
	}

	if opts.DataDirectory != "" {
		// Create parent directories if not exists
		if err := os.MkdirAll(opts.DataDirectory, 0755); err != nil {
			return pderr.Wrap("RunTestingPostgresContainer: failed to create data directory", err)
		}
	}

	if opts.UnixSocketPath != "" {
		// Create parent directories if not exists
		if err := os.MkdirAll(opts.UnixSocketPath, 0755); err != nil {
			return pderr.Wrap("RunTestingPostgresContainer: failed to create unix socket directory", err)
		}
	}

	argv := []string{
		opts.DockerBinary,
		"run",
		"--rm",
		"--name", opts.ContainerName,
		"-e", "POSTGRES_DB=" + opts.DatabaseName,
		"-e", "POSTGRES_USER=" + opts.Username,
		"-e", "POSTGRES_PASSWORD=" + opts.Password,
	}

	if useTmpfs {
		logger.Info("using tmpfs for data; database will be transient")
		argv = append(argv, "--mount", "type=tmpfs,target=/var/lib/postgresql/data")
	} else {
		logger.Info("using directory for data; database will be persistent", zap.String("data_directory", opts.DataDirectory))
		argv = append(argv, "--mount", "type=bind,source="+opts.DataDirectory+",target=/var/lib/postgresql/data")
	}

	argv = append(argv, []string{
		"--mount", "type=bind,source=" + opts.UnixSocketPath + ",target=/var/run/postgresql",
		opts.PostgresImage,
		"-c", "unix_socket_directories=/var/run/postgresql",
		"-c", "listen_addresses=",
	}...)

	logger.Info("running testing postgres container", zap.Strings("argv", argv))

	socketPath, err := filepath.Abs(opts.UnixSocketPath)
	if err != nil {
		return pderr.Wrap("RunTestingPostgresContainer: failed to get absolute path", err)
	}

	dbConnectInfo := DatabaseConnectionInfo{
		UnixSocketPath: socketPath,
		Username:       opts.Username,
		DatabaseName:   opts.DatabaseName,
	}

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var commandFinishedOK bool

	go func() {
		logger.Info("waiting for postgres to be ready")
		time.Sleep(5 * time.Second) // TODO: wait for postgres to be ready
		logger.Info("assuming postgres is ready")

		if runWithDatabase != nil {
			if err := runWithDatabase(ctx, dbConnectInfo); err != nil {
				logger.Error("failed to run with database", zap.Error(err))
			}

			commandFinishedOK = true
			cancel()
		}
	}()

	if err := cmd.Run(); err != nil && !commandFinishedOK {
		return pderr.Wrap("RunTestingPostgresContainer: failed to run command", err)
	}

	logger.Info("done running testing postgres container")

	return nil
}
