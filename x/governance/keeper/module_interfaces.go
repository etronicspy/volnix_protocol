package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	
	anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"
	consensustypes "github.com/volnix-protocol/volnix-protocol/x/consensus/types"
	lizenztypes "github.com/volnix-protocol/volnix-protocol/x/lizenz/types"
)

// LizenzKeeperForGovernance defines the interface for lizenz keeper used by governance
type LizenzKeeperForGovernance interface {
	GetParams(ctx sdk.Context) lizenztypes.Params
	SetParams(ctx sdk.Context, params lizenztypes.Params)
}

// AnteilKeeperForGovernance defines the interface for anteil keeper used by governance
type AnteilKeeperForGovernance interface {
	GetParams(ctx sdk.Context) anteiltypes.Params
	SetParams(ctx sdk.Context, params anteiltypes.Params)
}

// ConsensusKeeperForGovernance defines the interface for consensus keeper used by governance
type ConsensusKeeperForGovernance interface {
	GetParams(ctx sdk.Context) consensustypes.Params
	SetParams(ctx sdk.Context, params consensustypes.Params)
}

