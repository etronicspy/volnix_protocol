package types

const (
	EventTypeProposalSubmitted = "governance.proposal_submitted"

	EventTypeVoteCast = "governance.vote_cast"

	EventTypeProposalExecuted = "governance.proposal_executed"

	EventTypeProposalRejected = "governance.proposal_rejected"

	EventTypeParameterChanged = "governance.parameter_changed"

	AttributeKeyProposalID    = "proposal_id"
	AttributeKeyProposer      = "proposer"
	AttributeKeyVoter         = "voter"
	AttributeKeyVoteOption    = "vote_option"
	AttributeKeyProposalType  = "proposal_type"
	AttributeKeyResult        = "result"
	AttributeKeyBlockHeight   = "block_height"
	AttributeKeyModuleName    = "module_name"
	AttributeKeyParameterKey  = "parameter_key"
	AttributeKeyParameterValue = "parameter_value"
)
