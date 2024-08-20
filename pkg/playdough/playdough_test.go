package playdough

import (
	"context"
	"testing"

	"github.com/steinarvk/playdough/proto/pdpb"
)

func newForTesting(t *testing.T) *Playdough {
	pd, err := New(Params{})
	if err != nil {
		t.Fatalf("fatal error setting up Playdough for testing: %v", err)
	}
	return pd
}

func TestMe(t *testing.T) {
	_, err := newForTesting(t).CreateAccount(context.Background(), &pdpb.CreateAccountRequest{
		Username: "alice",
		Password: "hunter2",
	})
	if err != nil {
		t.Errorf("oops: %v", err)
	}
}
