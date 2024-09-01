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
	logger := logging.FromContext(ctx)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, pderr.Unexpectedf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	user, err := s.userdb.RegisterUserWithPassword(ctx, tx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, pderr.Unexpectedf("failed to commit transaction: %v", err)
	}

	logger.Info("created account with password", zap.String("username", user.Username), zap.Stringer("user_uuid", user.UserUUID))

	return &pdpb.CreateAccountResponse{
		Username: user.Username,
		UserUuid: user.UserUUID.String(),
	}, nil
}

func (s *server) Ping(ctx context.Context, req *pdpb.PingRequest) (*pdpb.PingResponse, error) {
	logger := logging.FromContext(ctx)
	logger.Info("processing ping request", zap.String("echo_message", req.Echo))

	return &pdpb.PingResponse{
		EchoResponse: req.Echo,
	}, nil
}

func (s *server) Login(ctx context.Context, req *pdpb.LoginRequest) (*pdpb.LoginResponse, error) {
	logger := logging.FromContext(ctx)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, pderr.Unexpectedf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	user, err := s.userdb.AuthenticateByPassword(ctx, tx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, pderr.Unexpectedf("failed to commit transaction: %v", err)
	}

	token, err := s.auth.IssueAuthenticatedToken(ctx, user.Username, 24*time.Hour)
	if err != nil {
		return nil, pderr.Unexpectedf("failed to issue token: %v", err)
	}

	logger.Info("logged in account with password", zap.String("username", user.Username), zap.Stringer("user_uuid", user.UserUUID))

	return &pdpb.LoginResponse{
		SessionToken: token,
		UserUuid:     user.UserUUID.String(),
	}, nil
}
