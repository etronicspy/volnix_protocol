package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "consensus",
		Short:                      "Consensus transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdDeclarePerHeightBurn())
	cmd.AddCommand(CmdRegisterConsensusMapping())

	return cmd
}

func CmdDeclarePerHeightBurn() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "declare-per-height-burn [s_i] [b_i]",
		Short: "Declare per-height burn amounts (priority stake and fee-share burn)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}

func CmdRegisterConsensusMapping() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-consensus-mapping [consensus-pubkey-hex]",
		Short: "Register consensus pubkey to account address mapping",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}
