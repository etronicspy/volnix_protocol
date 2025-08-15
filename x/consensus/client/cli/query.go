package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "consensus",
		Short:                      "Querying commands for the consensus module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// Add query commands here when they are implemented
	// cmd.AddCommand(CmdGetParams())
	// cmd.AddCommand(CmdGetValidators())

	return cmd
}
