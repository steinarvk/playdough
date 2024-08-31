package pdserver

import (
	"context"
	"time"

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

func (s *server) Login(ctx context.Context, req *pdpb.LoginRequest) (*pdpb.LoginResponse, error) {
	if req.Username == "testuser" && req.Password == "testpassword" {
		token, err := s.auth.IssueAuthenticatedToken(ctx, req.Username, 24*time.Hour)
		if err != nil {
			return nil, pderr.Unexpectedf("failed to issue token: %v", err)
		}

		return &pdpb.LoginResponse{
			SessionToken: token,
		}, nil
	}

	return nil, pderr.NotImplemented("CreateAccount: not implemented")
}
