package pdclient

import (
	"bytes"
	"context"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/steinarvk/playdough/pkg/pderr"
	"github.com/steinarvk/playdough/proto/pdpb"
	"golang.org/x/term"
	"google.golang.org/grpc/codes"
)

func makeSubcommands() []*Subcommand {
	return []*Subcommand{
		makeCreateAccountSubcommand(),
		makePingSubcommand(),
	}
}

func makeCreateAccountSubcommand() *Subcommand {
	cmd := cobra.Command{
		Use:   "create-account",
		Short: "create an account",
	}

	var params CreateAccountParams
	cmd.Flags().StringVar(&params.Username, "username", "", "username of the account to create")

	return &Subcommand{
		Command: &cmd,
		Core: func(ctx context.Context, client *Client) error {
			if params.Username == "" {
				return pderr.MissingRequiredFlag("--username")
			}
			fmt.Printf("Password for new user %q: ", params.Username)
			passwordOnce, err := term.ReadPassword(syscall.Stdin)
			if err != nil {
				return pderr.Wrap("error reading password from stdin", err)
			}
			fmt.Println()

			fmt.Printf("Repeat password for new user %q: ", params.Username)
			passwordTwice, err := term.ReadPassword(syscall.Stdin)
			if err != nil {
				return pderr.Wrap("error re-reading password from stdin", err)
			}
			fmt.Println()

			if !bytes.Equal(passwordOnce, passwordTwice) {
				return pderr.Error(codes.InvalidArgument, "passwords do not match")
			}

			if len(passwordOnce) == 0 {
				return pderr.Error(codes.InvalidArgument, "password cannot be empty")
			}

			password := string(passwordOnce)

			req := &pdpb.CreateAccountRequest{
				Username: params.Username,
				Password: password,
			}

			if _, err := client.grpcClient.CreateAccount(ctx, req); err != nil {
				return err
			}

			return err
		},
	}
}

func makePingSubcommand() *Subcommand {
	cmd := cobra.Command{
		Use:   "ping",
		Short: "send a ping message to test the connection",
	}

	var echoMessage string
	cmd.Flags().StringVar(&echoMessage, "message", "Hello world!", "message to be echoed in the response")

	return &Subcommand{
		Command: &cmd,
		Core: func(ctx context.Context, client *Client) error {
			req := &pdpb.PingRequest{
				Echo: echoMessage,
			}
			resp, err := client.grpcClient.Ping(ctx, req)
			if err != nil {
				return err
			}

			fmt.Printf("Pong: %s\n", resp.EchoResponse)
			return nil
		},
	}
}
