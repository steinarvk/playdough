package pdserver

import (
	"database/sql"

	"github.com/steinarvk/playdough/pkg/pdauth"
	"github.com/steinarvk/playdough/pkg/pddb/userdb"
	"github.com/steinarvk/playdough/proto/pdpb"
)

type server struct {
	pdpb.UnsafePlaydoughServiceServer

	db     *sql.DB
	auth   *pdauth.AuthValidator
	userdb *userdb.UserDB
}
