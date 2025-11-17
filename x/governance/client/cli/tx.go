package cli

import (
	"github.com/spf13/cobra"
)

// GetTxCmd returns the transaction commands for the governance module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "governance",
		Short: "Governance transaction subcommands",
		Long:  "Governance transaction subcommands for submitting proposals and voting",
	}
	
	// TODO: Add subcommands after proto generation
	// cmd.AddCommand(CmdSubmitProposal())
	// cmd.AddCommand(CmdVote())
	// cmd.AddCommand(CmdExecuteProposal())
	
	return cmd
}

