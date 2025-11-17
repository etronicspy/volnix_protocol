package cli

import (
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the query commands for the governance module
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "governance",
		Short: "Querying commands for the governance module",
		Long:  "Querying commands for proposals, votes, and governance parameters",
	}
	
	// TODO: Add subcommands after proto generation
	// cmd.AddCommand(CmdQueryProposal())
	// cmd.AddCommand(CmdQueryVotes())
	// cmd.AddCommand(CmdQueryParams())
	
	return cmd
}

