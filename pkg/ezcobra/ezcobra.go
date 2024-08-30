package ezcobra

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steinarvk/playdough/pkg/pderr"
)

func HandleErrors(core func(*cobra.Command, []string) error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if err := core(cmd, args); err != nil {
			pderr.HandleFatalAndDie(err)
		}
	}
}

func RunENoArgs(core func(ctx context.Context) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		if len(args) != 0 {
			return pderr.BadInput("invalid number of command-line arguments (expected zero)", "command-line-arguments", fmt.Sprintf("%v", args))
		}

		return core(ctx)
	}
}

func RunNoArgs(core func(ctx context.Context) error) func(*cobra.Command, []string) {
	return HandleErrors(RunENoArgs(core))
}
