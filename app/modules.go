package app

import (
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/volnix-protocol/volnix-protocol/x/anteil"
	"github.com/volnix-protocol/volnix-protocol/x/ident"
	"github.com/volnix-protocol/volnix-protocol/x/lizenz"
)

// ModuleBasics defines the module basic manager for Volnix app containing
// only custom modules at this stage.
var ModuleBasics = module.NewBasicManager(
	ident.AppModuleBasic{},
	lizenz.AppModuleBasic{},
	anteil.AppModuleBasic{},
)
