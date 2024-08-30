package pdauth

import (
	"context"

	"github.com/steinarvk/playdough/pkg/pderr"
)

type contextKey string

const (
	contextKeyAuthInfo contextKey = "authinfo"
)

type AuthInfo struct {
	IsAuthenticated       bool
	AuthenticatedUsername string
}

var (
	notAuthenticated = AuthInfo{
		IsAuthenticated:       false,
		AuthenticatedUsername: "",
	}
)

type AuthValidator struct {
}

func NewValidator() *AuthValidator {
	return &AuthValidator{}
}

func (a AuthValidator) ValidateHeader(headerValue string) (AuthInfo, error) {
	if headerValue == "" {
		return notAuthenticated, nil
	}

	if headerValue == "hunter2" {
		return AuthInfo{
			AuthenticatedUsername: "testuser",
		}, nil
	}

	return notAuthenticated, pderr.Unauthenticated("not accepted testing auth")
}

func NewContextWithAuth(ctx context.Context, authInfo AuthInfo) context.Context {
	return context.WithValue(ctx, contextKeyAuthInfo, authInfo)
}

func FromContext(ctx context.Context) AuthInfo {
	authInfo, ok := ctx.Value(contextKeyAuthInfo).(AuthInfo)
	if !ok {
		panic("no auth info in context")
	}
	return authInfo
}
