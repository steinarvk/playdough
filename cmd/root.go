package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func makeRootCmd() *cobra.Command {
	run := func(cmd *cobra.Command, args []string) error {
		fmt.Println("Hello world!")
		return nil
	}

	rootCmd := &cobra.Command{
		Use:   "playdough",
		Short: "centralized ledger for toy currencies",
		RunE:  run,
	}

	return rootCmd
}

func Execute() {
	if err := makeRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
