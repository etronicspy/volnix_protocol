package consensus

import (
	"github.com/volnix-protocol/volnix-protocol/x/consensus/types"
	"github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
)

// AppModuleBasic defines the basic application module used by the consensus module.
type AppModuleBasic struct{}

// Name returns the consensus module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// AppModule implements an application module for the consensus module.
type AppModule struct {
	AppModuleBasic

	keeper *keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(keeper *keeper.Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         keeper,
	}
}

// ConsensusVersion implements ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// IsAppModule implements module.AppModule
func (AppModule) IsAppModule() {}

// IsOnePerModuleType implements module.AppModule
func (AppModule) IsOnePerModuleType() {}

// DefaultGenesis returns default genesis state
func DefaultGenesis() *types.GenesisState {
	return types.DefaultGenesis()
}
