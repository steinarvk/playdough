package userdb

import (
	"regexp"

	"github.com/steinarvk/playdough/pkg/pderr"
	"google.golang.org/grpc/codes"
)

var (
	validUsernameRE = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
)

func CheckValidUsername(username string) error {
	if !validUsernameRE.MatchString(username) {
		return pderr.Error(codes.InvalidArgument, "invalid username")
	}

	return nil
}

var (
	minPasswordLength = 8
)

func CheckValidPassword(password string) error {
	if len(password) < minPasswordLength {
		return pderr.Error(codes.InvalidArgument, "password too short")
	}

	return nil
}
