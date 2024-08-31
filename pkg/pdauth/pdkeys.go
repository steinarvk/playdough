package pdauth

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/steinarvk/playdough/pkg/logging"
	"github.com/steinarvk/playdough/pkg/pderr"
	"go.uber.org/zap"
)

const (
	activeHMACAlgorithm = "HS256"
	keyActiveDuration   = 24 * time.Hour
)

type SigningKey struct {
	KeyUUID           uuid.UUID
	KeyAlgorithmName  string
	KeySecretData     []byte
	KeyGenerationTime time.Time
	KeyExpirationTime time.Time
}

func (a *AuthValidator) holdingMutexGetKeyAlgorithmId(ctx context.Context, algorithmName string) (int, error) {
	rv, ok := a.cachedAlgorithmIDs[algorithmName]
	if ok {
		return rv, nil
	}

	var algorithmID int

	err := a.db.QueryRowContext(
		ctx,
		`
			SELECT jwt_key_algorithm_id
			FROM jwt_key_algorithms
			WHERE algorithm_name = $1
		`,
		algorithmName,
	).Scan(&algorithmID)
	if err == nil {
		a.cachedAlgorithmIDs[algorithmName] = algorithmID
		return algorithmID, nil
	}

	if err != sql.ErrNoRows {
		return 0, err
	}

	err = a.db.QueryRowContext(
		ctx,
		`
			INSERT INTO jwt_key_algorithms (algorithm_name)
			VALUES ($1)
			RETURNING
				jwt_key_algorithm_id
		`,
		algorithmName,
	).Scan(&algorithmID)
	if err != nil {
		return 0, err
	}

	a.cachedAlgorithmIDs[algorithmName] = algorithmID
	return algorithmID, nil
}

func (a *AuthValidator) holdingMutexGenerateSigningKey(ctx context.Context, now time.Time) (*SigningKey, error) {
	logger := logging.FromContext(ctx)

	randomBytes := make([]byte, 32)
	if _, err := cryptorand.Read(randomBytes); err != nil {
		return nil, err
	}

	keyUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	logger.Info(
		"generated new JWT signing key",
		zap.Stringer("key_uuid", keyUUID),
		zap.String("algorithm", activeHMACAlgorithm),
		zap.Time("generation_time", now),
		zap.Time("expiration_time", now.Add(keyActiveDuration)),
		zap.Int("key_length", len(randomBytes)),
	)

	return &SigningKey{
		KeyUUID:           keyUUID,
		KeyAlgorithmName:  activeHMACAlgorithm,
		KeySecretData:     randomBytes,
		KeyGenerationTime: now,
		KeyExpirationTime: now.Add(keyActiveDuration),
	}, nil
}

func (a *AuthValidator) getValidationKey(ctx context.Context, keyUUID uuid.UUID) (*SigningKey, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if cachedKey, ok := a.cachedValidationKeys[keyUUID]; ok {
		return cachedKey, nil
	}

	var key SigningKey

	err := a.db.QueryRowContext(
		ctx,
		`
			SELECT
				jwt_keys.jwt_key_uuid,
				jwt_key_algorithms.algorithm_name,
				jwt_keys.key_secret_material,
				jwt_keys.generation_timestamp,
				jwt_keys.expiration_timestamp
			FROM jwt_keys
			LEFT JOIN jwt_key_algorithms ON jwt_keys.jwt_key_algorithm_id = jwt_key_algorithms.jwt_key_algorithm_id
			WHERE jwt_keys.jwt_key_uuid = $1
		`,
		keyUUID,
	).Scan(&key.KeyUUID, &key.KeyAlgorithmName, &key.KeySecretData, &key.KeyGenerationTime, &key.KeyExpirationTime)

	if err == sql.ErrNoRows {
		logger := logging.FromContext(ctx)
		logger.Warn("unknown key ID", zap.Stringer("key_uuid", keyUUID))
		return nil, pderr.BadInput("unknown key ID", "key_uuid", keyUUID.String())
	}

	if err != nil {
		return nil, err
	}

	a.cachedValidationKeys[keyUUID] = &key

	return &key, nil
}

func (a *AuthValidator) getActiveSigningKey(ctx context.Context) (*SigningKey, error) {
	now := time.Now()

	a.mu.Lock()
	defer a.mu.Unlock()

	return a.holdingMutexGetActiveSigningKey(ctx, now)
}

func (a *AuthValidator) holdingMutexGetActiveSigningKey(ctx context.Context, now time.Time) (*SigningKey, error) {
	if a.cachedSigningKey != nil && a.cachedSigningKey.KeyExpirationTime.After(now) {
		return a.cachedSigningKey, nil
	}

	var signingKey SigningKey

	err := a.db.QueryRowContext(
		ctx,
		`
			SELECT
				jwt_keys.jwt_key_uuid,
				jwt_key_algorithms.algorithm_name,
				jwt_keys.key_secret_material,
				jwt_keys.generation_timestamp,
				jwt_keys.expiration_timestamp
			FROM jwt_keys
			LEFT JOIN jwt_key_algorithms ON jwt_keys.jwt_key_algorithm_id = jwt_key_algorithms.jwt_key_algorithm_id
			WHERE jwt_keys.expiration_timestamp > $1
			ORDER BY generation_timestamp DESC
			LIMIT 1
		`,
		now,
	).Scan(&signingKey.KeyUUID, &signingKey.KeyAlgorithmName, &signingKey.KeySecretData, &signingKey.KeyGenerationTime, &signingKey.KeyExpirationTime)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == nil {
		a.cachedSigningKey = &signingKey
		return &signingKey, nil
	}

	newSigningKey, err := a.holdingMutexGenerateSigningKey(ctx, now)
	if err != nil {
		return nil, err
	}

	algoID, err := a.holdingMutexGetKeyAlgorithmId(ctx, newSigningKey.KeyAlgorithmName)
	if err != nil {
		return nil, err
	}

	if _, err := a.db.ExecContext(
		ctx,
		`
			INSERT INTO jwt_keys (jwt_key_uuid, jwt_key_algorithm_id, key_secret_material, generation_timestamp, expiration_timestamp)
			VALUES ($1, $2, $3, $4, $5)
		`,
		newSigningKey.KeyUUID, algoID, newSigningKey.KeySecretData, newSigningKey.KeyGenerationTime, newSigningKey.KeyExpirationTime,
	); err != nil {
		return nil, err
	}

	return newSigningKey, nil
}
