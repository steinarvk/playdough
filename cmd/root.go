package cmd

import (
	"github.com/spf13/cobra"
	"github.com/steinarvk/playdough/pkg/pdclient"
	"github.com/steinarvk/playdough/pkg/pderr"
	"github.com/steinarvk/playdough/pkg/pdservermain"
	"github.com/steinarvk/playdough/pkg/pdtestutils"
)

func makeServeCmd() *cobra.Command {
	return pdservermain.NewCobraCommand()
}

func makeRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "playdough",
		Short: "centralized ledger for toy currencies",
	}

	rootCmd.AddCommand(makeServeCmd())
	rootCmd.AddCommand(pdclient.MakeCobraCommandGroup())
	rootCmd.AddCommand(pdtestutils.MakeTestingCommandGroup())

	return rootCmd
}

func Execute() {
	pderr.HandleFatalAndDie(makeRootCmd().Execute())
}
