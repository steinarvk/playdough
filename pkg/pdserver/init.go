package pdserver

import (
	"github.com/steinarvk/playdough/proto/pdpb"
)

type Option func(*server) error

func (s *server) finalize() error {
	return nil
}

func New(options ...Option) (pdpb.PlaydoughServiceServer, error) {
	rv := &server{}
	for _, opt := range options {
		if err := opt(rv); err != nil {
			return nil, err
		}
	}
	if err := rv.finalize(); err != nil {
		return nil, err
	}
	return rv, nil
}
