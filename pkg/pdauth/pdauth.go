package pdauth

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/steinarvk/playdough/pkg/logging"
	"github.com/steinarvk/playdough/pkg/pderr"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type contextKey string

const (
	contextKeyAuthInfo contextKey = "authinfo"

	jwtIssuer = "playdough"
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
	mu sync.Mutex

	db                   *sql.DB
	cachedAlgorithmIDs   map[string]int
	cachedSigningKey     *SigningKey
	cachedValidationKeys map[uuid.UUID]*SigningKey
}

func NewValidator(db *sql.DB) *AuthValidator {
	return &AuthValidator{
		db:                   db,
		cachedValidationKeys: map[uuid.UUID]*SigningKey{},
		cachedAlgorithmIDs:   map[string]int{},
	}
}

const (
	usernamePrefix = "u:"
)

func (a *AuthValidator) IssueAuthenticatedToken(ctx context.Context, authenticatedUsername string, validDuration time.Duration) (string, error) {
	key, err := a.getActiveSigningKey(ctx)
	if err != nil {
		return "", err
	}

	tokenUUID, err := uuid.NewRandom()
	if err != nil {
		return "", pderr.Unexpectedf("failed to generate token UUID")
	}

	issuedTime := time.Now()
	expiresTime := issuedTime.Add(validDuration)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(issuedTime),
		ExpiresAt: jwt.NewNumericDate(expiresTime),
		Issuer:    jwtIssuer,
		Subject:   usernamePrefix + authenticatedUsername,
		ID:        tokenUUID.String(),
	})
	token.Header["kid"] = key.KeyUUID.String()

	tokenString, err := token.SignedString(key.KeySecretData)
	if err != nil {
		return "", pderr.Unexpectedf("failed to sign token")
	}

	logger := logging.FromContext(ctx)
	logger.Info("issued token",
		zap.String("username", authenticatedUsername),
		zap.Time("issued_at", issuedTime),
		zap.Time("expires_at", expiresTime),
		zap.Stringer("token_id", tokenUUID),
		zap.Stringer("key_id", key.KeyUUID),
		zap.String("alg", token.Method.Alg()),
	)

	return tokenString, nil
}

func (a *AuthValidator) ValidateHeader(ctx context.Context, headerValue string) (AuthInfo, error) {
	if headerValue == "" {
		return notAuthenticated, nil
	}

	components := strings.SplitN(headerValue, " ", 2)
	if len(components) != 2 {
		return notAuthenticated, pderr.Unauthenticated("malformed auth header")
	}

	if components[0] != "Bearer" {
		return notAuthenticated, pderr.Unauthenticated("unsupported auth scheme (not Bearer)")
	}

	tokenString := components[1]

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != activeHMACAlgorithm {
			return nil, pderr.Unauthenticated("unsupported signing algorithm")
		}

		keyUUIDValue, ok := token.Header["kid"]
		if !ok {
			return nil, pderr.Unauthenticated("missing key ID")
		}

		keyUUIDString, ok := keyUUIDValue.(string)
		if !ok {
			return nil, pderr.Unauthenticated("kid is not string")
		}

		keyUUID, err := uuid.Parse(keyUUIDString)
		if err != nil {
			return nil, pderr.Unauthenticated("invalid key ID")
		}

		validationKey, err := a.getValidationKey(ctx, keyUUID)
		if err != nil {
			return nil, err
		}

		return validationKey.KeySecretData, nil
	}, jwt.WithExpirationRequired(), jwt.WithIssuer(jwtIssuer))
	if err != nil {
		return notAuthenticated, pderr.WrapAs(codes.Unauthenticated, "token validation failed", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return notAuthenticated, pderr.Unauthenticated("invalid token")
	}

	subject, ok := claims["sub"].(string)
	if !ok {
		return notAuthenticated, pderr.Unauthenticated("missing subject")
	}

	if username, ok := strings.CutPrefix(subject, usernamePrefix); ok {
		return AuthInfo{
			IsAuthenticated:       true,
			AuthenticatedUsername: username,
		}, nil
	} else {
		return notAuthenticated, pderr.Unauthenticated("invalid subject")
	}
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
