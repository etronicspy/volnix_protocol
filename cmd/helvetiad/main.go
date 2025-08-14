package main

import (
	"fmt"
	"os"

	sdklog "cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	apppkg "github.com/helvetia-protocol/helvetia-protocol/app"
)

// Application version and git commit. Commit is injected via -ldflags at build time.
var (
	appVersion = "0.1.0"
	commit     = "dev"
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "helvetiad",
		Short:         "Helvetia Protocol daemon",
		Long:          "Helvetia Protocol (H•P) — sovereign L1 blockchain on Cosmos SDK. Bootstrap daemon.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newStartCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print helvetiad version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("helvetiad %s (%s)\n", appVersion, commit)
		},
	}
	return cmd
}

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Helvetia node (init app stores in-memory)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bech32 prefixes
			cfg := sdk.GetConfig()
			cfg.SetBech32PrefixForAccount("hp", "hppub")
			cfg.SetBech32PrefixForValidator("hpvaloper", "hpvaloperpub")
			cfg.SetBech32PrefixForConsensusNode("hpvalcons", "hpvalconspub")
			cfg.Seal()

			// Encoding and in-memory DB
			encoding := apppkg.MakeEncodingConfig()
			logger := sdklog.NewNopLogger()
			database := dbm.NewMemDB()

			// Build app and load latest version
			hpApp := apppkg.NewHelvetiaApp(logger, database, nil, encoding)
			if err := hpApp.LoadLatestVersion(); err != nil {
				return err
			}

			fmt.Println("Helvetia app initialized in-memory. ABCI/Tendermint server wiring will be added later.")
			return nil
		},
	}
	return cmd
}
