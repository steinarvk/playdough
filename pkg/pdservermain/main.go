package pdservermain

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/spf13/cobra"
	"github.com/steinarvk/playdough/pkg/ezcobra"
	"github.com/steinarvk/playdough/pkg/logging"
	"github.com/steinarvk/playdough/pkg/pderr"
	"github.com/steinarvk/playdough/pkg/pdserver"
	"github.com/steinarvk/playdough/proto/pdpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	defaultListenPort = 5044
)

type ListenAddress struct {
	Host string
	Port int
}

type Params struct {
	ListenAddress ListenAddress
}

func NewCobraCommand() *cobra.Command {
	var params Params

	rv := &cobra.Command{
		Use:   "serve",
		Short: "run a PlayDoughService gRPC server",
		Run: ezcobra.RunNoArgs(func(ctx context.Context) error {
			return Main(ctx, params)
		}),
	}

	rv.Flags().StringVar(&params.ListenAddress.Host, "host", "localhost", "address on which to listen")
	rv.Flags().IntVar(&params.ListenAddress.Port, "port", defaultListenPort, "port on which to listen")

	return rv
}

func Main(ctx context.Context, params Params) error {
	logger, err := zap.NewProduction()
	if err != nil {
		return pderr.Unexpectedf("failed to initialize logging with zap: %w", err)
	}

	listenHost := params.ListenAddress.Host
	if listenHost == "" {
		listenHost = "localhost"
	}

	listenPort := params.ListenAddress.Port
	if listenPort == 0 {
		listenPort = defaultListenPort
	}

	pdServer, err := pdserver.New()
	if err != nil {
		return err
	}

	listenAddr := fmt.Sprintf("%s:%d", listenHost, listenPort)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	var opts []grpc.ServerOption

	opts = append(opts, grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		t0 := time.Now()

		sublogger := logger.With(zap.String("method", info.FullMethod))

		debugMode := false

		ctx = logging.NewContextWithLogger(ctx, sublogger, debugMode)

		sublogger.Info("incoming gRPC request")

		resp, err := handler(ctx, req)

		duration := time.Since(t0)
		durationField := zap.Duration("duration", duration)

		if err != nil {
			sublogger.Warn("gRPC request error", durationField, zap.Stringer("code", pderr.CodeOf(err)), zap.Error(err))
		} else {
			sublogger.Info("gRPC request finished", durationField)
		}

		return resp, err
	}))

	grpcServer := grpc.NewServer(opts...)
	pdpb.RegisterPlaydoughServiceServer(grpcServer, pdServer)

	logger.Info("ready to serve gRPC (PlaydoughService)", zap.String("listen_addr", listenAddr))

	if err := grpcServer.Serve(listener); err != nil {
		return pderr.Wrap("gRPC Serve() error", err)
	}

	logger.Info("finished serving gRPC (PlaydoughService)", zap.String("listen_addr", listenAddr))

	return nil
}
