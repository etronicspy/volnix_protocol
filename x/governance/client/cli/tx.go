package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	governancev1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/governance/v1"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "governance",
		Short: "Governance transaction subcommands",
		Long:  "Governance transaction subcommands for submitting proposals and voting",
	}

	cmd.AddCommand(CmdSubmitProposal())
	cmd.AddCommand(CmdVote())

	return cmd
}

func CmdSubmitProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-proposal [title] [description]",
		Short: "Submit a new governance proposal",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposalTypeStr, _ := cmd.Flags().GetString("type")

			var proposalType governancev1.ProposalType
			switch proposalTypeStr {
			case "parameter_change":
				proposalType = governancev1.ProposalType_PROPOSAL_TYPE_PARAMETER_CHANGE
			case "text":
				proposalType = governancev1.ProposalType_PROPOSAL_TYPE_TEXT
			default:
				proposalType = governancev1.ProposalType_PROPOSAL_TYPE_PARAMETER_CHANGE
			}

			msg := &governancev1.MsgSubmitProposal{
				Proposer:     clientCtx.GetFromAddress().String(),
				Title:        args[0],
				Description:  args[1],
				ProposalType: proposalType,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("type", "parameter_change", "Proposal type (parameter_change, text)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdVote() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote [proposal-id] [option]",
		Short: "Vote on a proposal (yes/no/abstain)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposalID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid proposal ID: %w", err)
			}

			var voteOption governancev1.VoteOption
			switch args[1] {
			case "yes":
				voteOption = governancev1.VoteOption_VOTE_OPTION_YES
			case "no":
				voteOption = governancev1.VoteOption_VOTE_OPTION_NO
			case "abstain":
				voteOption = governancev1.VoteOption_VOTE_OPTION_ABSTAIN
			default:
				return fmt.Errorf("invalid vote option: %s (use yes/no/abstain)", args[1])
			}

			msg := &governancev1.MsgVote{
				Voter:      clientCtx.GetFromAddress().String(),
				ProposalId: proposalID,
				Option:     voteOption,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
