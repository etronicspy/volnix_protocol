package main

import (
	"fmt"
	"os"

	sdklog "cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	apppkg "github.com/volnix-protocol/volnix-protocol/app"
)

// Application version and git commit. Commit is injected via -ldflags at build time.
var (
	appVersion = "0.1.0"
	commit     = "dev"
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "volnixd",
		Short:         "Volnix Protocol daemon",
		Long:          "Volnix Protocol — sovereign L1 blockchain on Cosmos SDK. Bootstrap daemon.",
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
		Short: "Print volnixd version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("volnixd %s (%s)\n", appVersion, commit)
		},
	}
	return cmd
}

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Volnix node (init app stores in-memory)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bech32 prefixes
			cfg := sdk.GetConfig()
			cfg.SetBech32PrefixForAccount("vx", "vxpub")
			cfg.SetBech32PrefixForValidator("vxvaloper", "vxvaloperpub")
			cfg.SetBech32PrefixForConsensusNode("vxvalcons", "vxvalconspub")
			cfg.Seal()

			// Encoding and in-memory DB
			encoding := apppkg.MakeEncodingConfig()
			logger := sdklog.NewNopLogger()
			database := dbm.NewMemDB()

			// Build app and load latest version
			app := apppkg.NewVolnixApp(logger, database, nil, encoding)
			if err := app.LoadLatestVersion(); err != nil {
				return err
			}

			fmt.Println("Volnix app initialized in-memory. ABCI/Tendermint server wiring will be added later.")
			return nil
		},
	}
	return cmd
}
