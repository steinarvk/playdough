package pdserver

import (
	"database/sql"

	"github.com/steinarvk/playdough/pkg/pdauth"
	"github.com/steinarvk/playdough/pkg/pddb/userdb"
	"github.com/steinarvk/playdough/proto/pdpb"
)

type Option func(*server) error

func (s *server) finalize() error {
	return nil
}

func New(db *sql.DB, options ...Option) (pdpb.PlaydoughServiceServer, error) {
	rv := &server{
		db: db,
	}

	for _, opt := range options {
		if err := opt(rv); err != nil {
			return nil, err
		}
	}
	if err := rv.finalize(); err != nil {
		return nil, err
	}

	rv.auth = pdauth.NewValidator(db)
	rv.userdb = userdb.New(db)

	return rv, nil
}
