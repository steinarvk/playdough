package playdough

import (
	"context"

	"github.com/steinarvk/playdough/proto/pdpb"
)

type Playdough struct {
}

type Params struct {
}

func New(params Params) (*Playdough, error) {
	return &Playdough{}, nil
}

type Error struct {
	msg string
}

func (e Error) Error() string {
	return e.msg
}

func (p *Playdough) CreateAccount(ctx context.Context, req *pdpb.CreateAccountRequest) (*pdpb.CreateAccountResponse, error) {
	return nil, Error{"oops"}
}
