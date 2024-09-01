package userdb

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"database/sql"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/steinarvk/playdough/pkg/logging"
	"github.com/steinarvk/playdough/pkg/pderr"
	"github.com/steinarvk/playdough/proto/pdpb"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

type UserDB struct {
	db *sql.DB
}

type User struct {
	UserUUID uuid.UUID
	Username string
}

func New(db *sql.DB) *UserDB {
	return &UserDB{
		db: db,
	}
}

var (
	argon2Parallelism = 1

	saltSize = 16
)

func getActivePasswordHashingMethod() *pdpb.PasswordHashingMethod {
	return &pdpb.PasswordHashingMethod{
		Method: &pdpb.PasswordHashingMethod_Argon2{
			Argon2: &pdpb.Argon2Params{
				TimeCost:   2,
				MemoryCost: 64 * 1024,
				KeyLength:  32,
			},
		},
	}
}

func hashPasswordArgon2(password string, salt []byte, params *pdpb.Argon2Params) ([]byte, error) {
	hashedPassword := argon2.IDKey([]byte(password), salt, params.TimeCost, params.MemoryCost, uint8(argon2Parallelism), params.KeyLength)
	return hashedPassword, nil
}

func hashPassword(password string, salt []byte, method *pdpb.PasswordHashingMethod) ([]byte, error) {
	switch method := method.Method.(type) {
	case *pdpb.PasswordHashingMethod_Argon2:
		return hashPasswordArgon2(password, salt, method.Argon2)
	default:
		return nil, pderr.Unexpectedf("unsupported password hashing method")
	}
}

func isUniqueViolation(err error, constraintName string) bool {
	pgerr, ok := err.(*pq.Error)
	if !ok {
		return false
	}

	return pgerr.Code == pq.ErrorCode("23505") && pgerr.Constraint == constraintName
}

func (u *UserDB) RegisterUserWithPassword(ctx context.Context, tx *sql.Tx, username, password string) (*User, error) {
	logger := logging.FromContext(ctx)

	if err := CheckValidUsername(username); err != nil {
		return nil, err
	}

	if err := CheckValidPassword(password); err != nil {
		return nil, err
	}

	logger.Info("registering user", zap.String("username", username))

	hashingMethod := getActivePasswordHashingMethod()

	hashingMethodBytes, err := proto.Marshal(hashingMethod)
	if err != nil {
		return nil, pderr.Wrap("failed to marshal password hashing method", err)
	}

	salt := make([]byte, saltSize)
	if _, err := cryptorand.Read(salt); err != nil {
		return nil, pderr.Wrap("failed to generate salt", err)
	}

	userUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, pderr.Wrap("failed to generate user UUID", err)
	}

	hashedPassword, err := hashPassword(password, salt, hashingMethod)
	if err != nil {
		return nil, pderr.Wrap("failed to hash password", err)
	}

	var userID int

	if err := tx.QueryRowContext(
		ctx,
		`
			INSERT INTO users
				(user_uuid, username)
			VALUES
				($1, $2)
			RETURNING user_id
		`,
		userUUID, username,
	).Scan(&userID); err != nil {
		if isUniqueViolation(err, "users_username_key") {
			return nil, pderr.Error(codes.AlreadyExists, "username already exists")
		}
		return nil, pderr.Wrap("failed to insert user", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`
			INSERT INTO password_credentials
				(user_id,
				 hashing_method,
				 password_hash,
				 password_salt)
			VALUES
				($1, $2, $3, $4)
		`,
		userID, hashingMethodBytes, hashedPassword, salt,
	); err != nil {
		return nil, pderr.Wrap("failed to insert password credentials", err)
	}

	logger.Info("successfully registered user", zap.String("username", username))

	return &User{
		UserUUID: userUUID,
		Username: username,
	}, nil
}

func (u *UserDB) AuthenticateByPassword(ctx context.Context, tx *sql.Tx, username, password string) (*User, error) {
	logger := logging.FromContext(ctx)
	logger.Info("attempting to authenticate user by password", zap.String("username", username))

	var rv User

	var hashingMethodBytes []byte
	var hashedPassword []byte
	var passwordSalt []byte

	if err := tx.QueryRowContext(
		ctx,
		`
			SELECT
				users.user_uuid,
				users.username,
				password_credentials.hashing_method,
				password_credentials.password_hash,
				password_credentials.password_salt
			FROM users
			LEFT JOIN password_credentials ON users.user_id = password_credentials.user_id
			WHERE users.username = $1
		`,
		username,
	).Scan(&rv.UserUUID, &rv.Username, &hashingMethodBytes, &hashedPassword, &passwordSalt); err != nil {
		return nil, pderr.Wrap("failed to fetch user by username", err)
	}

	hashingMethod := &pdpb.PasswordHashingMethod{}
	if err := proto.Unmarshal(hashingMethodBytes, hashingMethod); err != nil {
		return nil, pderr.Wrap("failed to unmarshal password hashing method", err)
	}

	newlyHashedPassword, err := hashPassword(password, passwordSalt, hashingMethod)
	if err != nil {
		return nil, pderr.Wrap("failed to hash password", err)
	}

	if !bytes.Equal(newlyHashedPassword, hashedPassword) {
		return nil, pderr.Error(codes.Unauthenticated, "password mismatch")
	}

	return &rv, nil
}

func (u *UserDB) FetchUserByUsername(ctx context.Context, tx *sql.Tx, username string) (*User, error) {
	var rv User

	if err := tx.QueryRowContext(
		ctx,
		`
			SELECT user_uuid, username
			FROM users
			WHERE username = $1
		`,
	).Scan(&rv.UserUUID, &rv.Username); err != nil {
		return nil, pderr.Wrap("failed to fetch user by username", err)
	}

	return &rv, nil
}
