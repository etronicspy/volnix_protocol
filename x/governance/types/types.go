package types

// GenesisState defines the governance module genesis state
// Uses temporary types until proto generation
// Note: Proposal and Vote types are defined in keeper package to avoid circular imports
type GenesisState struct {
	Params    Params
	Proposals []interface{} // Will be []keeper.Proposal after import
	Votes     []interface{} // Will be []keeper.Vote after import
}

