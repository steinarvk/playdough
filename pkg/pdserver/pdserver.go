package pdserver

import (
	"context"

	"github.com/steinarvk/playdough/pkg/logging"
	"github.com/steinarvk/playdough/pkg/pderr"
	"github.com/steinarvk/playdough/proto/pdpb"
	"go.uber.org/zap"
)

func (s *server) CreateAccount(ctx context.Context, req *pdpb.CreateAccountRequest) (*pdpb.CreateAccountResponse, error) {
	return nil, pderr.NotImplemented("CreateAccount: not implemented")
}

func (s *server) Ping(ctx context.Context, req *pdpb.PingRequest) (*pdpb.PingResponse, error) {
	logger := logging.FromContext(ctx)
	logger.Info("processing ping request", zap.String("echo_message", req.Echo))

	return &pdpb.PingResponse{
		EchoResponse: req.Echo,
	}, nil
}
