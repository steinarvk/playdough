package pdserver

import (
	"github.com/steinarvk/playdough/proto/pdpb"
)

type server struct {
	pdpb.UnsafePlaydoughServiceServer

	// logger *zap.Logger
}
