package pdclient

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/steinarvk/playdough/pkg/ezcobra"
	"github.com/steinarvk/playdough/pkg/pderr"
	"github.com/steinarvk/playdough/proto/pdpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	logger           *zap.Logger
	conn             *grpc.ClientConn
	grpcClient       pdpb.PlaydoughServiceClient
	commonParams     *CommonParams
	outgoingMetadata metadata.MD
}

func (c *Client) OutgoingContext(ctx context.Context) context.Context {
	if c.outgoingMetadata == nil {
		return ctx
	}

	return metadata.NewOutgoingContext(ctx, c.outgoingMetadata)
}

func (c *Client) maybeDebugDump(methodName string, msgKind string, msg any) {
	if !c.commonParams.DebugMode {
		return
	}

	protoMsg, ok := msg.(proto.Message)
	if !ok {
		c.logger.Warn(
			"gRPC data is not proto.Message",
			zap.String("method", methodName),
			zap.String("kind", msgKind),
		)
		return
	}

	dataBytes, err := proto.Marshal(protoMsg)
	pderr.CheckOrPanic(err)

	marshaller := prototext.MarshalOptions{
		Multiline:    false,
		EmitASCII:    true,
		AllowPartial: true,
		EmitUnknown:  true,
	}

	marshalledMessage := marshaller.Format(protoMsg)

	c.logger.Info(
		"gRPC data",
		zap.String("method", methodName),
		zap.String("kind", msgKind),
		zap.String("message", marshalledMessage),
		zap.Int("length", len(dataBytes)),
	)
}

type CommonParams struct {
	ServerAddress           string
	DebugMode               bool
	InsecureGRPCCredentials bool
	RawAuthHeader           string
}

type CreateAccountParams struct {
	Username string
}

type Subcommand struct {
	Command *cobra.Command
	Core    func(context.Context, *Client) error
}

func addSubcommand(parent *cobra.Command, params *CommonParams, subcommand *Subcommand) {
	subcommand.Command.Run = ezcobra.RunNoArgs(func(ctx context.Context) error {
		return connectAndRunWithClient(ctx, params, subcommand.Core)
	})
	parent.AddCommand(subcommand.Command)
}

func makeLogger(_ *CommonParams) (*zap.Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, pderr.Wrap("failed to create zap logger", err)
	}

	return logger, nil
}

func connectAndRunWithClient(ctx context.Context, commonParams *CommonParams, core func(context.Context, *Client) error) error {
	client := &Client{
		commonParams: commonParams,
	}

	logger, err := makeLogger(commonParams)
	if err != nil {
		return err
	}

	client.logger = logger

	if commonParams.ServerAddress == "" {
		return pderr.MissingRequiredFlag("--server-address")
	}

	var opts []grpc.DialOption

	if commonParams.InsecureGRPCCredentials {
		client.logger.Warn("running with --insecure-grpc-credentials; don't do this in production")

		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if commonParams.RawAuthHeader != "" {
		client.outgoingMetadata = metadata.Pairs("Authorization", commonParams.RawAuthHeader)
	}

	opts = append(opts, grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		methodField := zap.String("method", method)

		if commonParams.DebugMode {
			client.logger.Info("sending gRPC request", methodField)
			client.maybeDebugDump(method, "request", req)
		}

		t0 := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(t0)
		durationField := zap.Duration("duration", duration)

		if commonParams.DebugMode {
			client.maybeDebugDump(method, "response", reply)

			client.logger.Info("finished gRPC request", methodField, durationField, zap.Bool("ok", err == nil), zap.Error(err))
		}

		if err != nil {
			err = pderr.WrapGRPCClient(method, err)
		}

		return err
	}))

	zapAddressField := zap.String("address", commonParams.ServerAddress)

	client.logger.Info("dialing gRPC", zapAddressField)
	conn, err := grpc.NewClient(commonParams.ServerAddress, opts...)
	if err != nil {
		return pderr.Wrap(fmt.Sprintf("failed to dial gRPC at %q", commonParams.ServerAddress), err)
	}
	client.conn = conn

	defer func() {
		client.logger.Info("closing gRPC connection", zapAddressField)
		conn.Close()
	}()

	pdClient := pdpb.NewPlaydoughServiceClient(conn)
	client.grpcClient = pdClient

	return core(ctx, client)
}

func MakeCobraCommandGroup() *cobra.Command {
	var params CommonParams

	group := &cobra.Command{
		Use:   "client",
		Short: "CLI client for PlaydoughService",
	}

	group.PersistentFlags().StringVar(&params.ServerAddress, "server-address", "localhost:5044", "address of PlayDoughService gRPC server")
	group.PersistentFlags().BoolVar(&params.DebugMode, "debug-dump-all", false, "dump all requests and responses for debugging")
	group.PersistentFlags().BoolVar(&params.InsecureGRPCCredentials, "insecure-grpc-credentials", false, "use insecure credentials")
	group.PersistentFlags().StringVar(&params.RawAuthHeader, "raw-auth-header", "", "raw authorization header")

	for _, subcommand := range makeSubcommands() {
		addSubcommand(group, &params, subcommand)
	}

	return group
}
