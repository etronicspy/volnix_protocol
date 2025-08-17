package ident

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	"github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"
)

// DefaultGenesis returns the default genesis state (protobuf type)
func DefaultGenesis() *identv1.GenesisState {
	return &identv1.GenesisState{
		Params:                 identtypes.DefaultParams().ToProto(),
		VerifiedAccounts:       []*identv1.VerifiedAccount{},
		IdentityVerifications:  []*identv1.IdentityVerification{},
		RoleMigrations:         []*identv1.RoleMigration{},
		VerificationProviders:  []*identv1.VerificationProvider{},
		ZkpProofs:              []*identv1.ZKPProof{},
	}
}

// Validate performs basic genesis state validation
func Validate(gen *identv1.GenesisState) error {
	if gen == nil {
		return identtypes.DefaultParams().Validate()
	}
	return identtypes.ParamsFromProto(gen.Params).Validate()
}

// InitGenesis initializes the module state from protobuf genesis data
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState *identv1.GenesisState) {
	if genState == nil {
		genState = DefaultGenesis()
	}
	k.SetParams(ctx, identtypes.ParamsFromProto(genState.Params))
	// accounts population would be implemented later
}

// ExportGenesis exports the current state as protobuf genesis
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *identv1.GenesisState {
	params := k.GetParams(ctx)
	return &identv1.GenesisState{
		Params:                 params.ToProto(),
		VerifiedAccounts:       []*identv1.VerifiedAccount{},
		IdentityVerifications:  []*identv1.IdentityVerification{},
		RoleMigrations:         []*identv1.RoleMigration{},
		VerificationProviders:  []*identv1.VerificationProvider{},
		ZkpProofs:              []*identv1.ZKPProof{},
	}
}
