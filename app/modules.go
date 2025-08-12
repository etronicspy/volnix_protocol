package app

import (
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/helvetia-protocol/helvetia-protocol/x/anteil"
	"github.com/helvetia-protocol/helvetia-protocol/x/ident"
	"github.com/helvetia-protocol/helvetia-protocol/x/lizenz"
)

// ModuleBasics defines the module basic manager for Helvetia app containing
// only custom modules at this stage.
var ModuleBasics = module.NewBasicManager(
	ident.AppModuleBasic{},
	lizenz.AppModuleBasic{},
	anteil.AppModuleBasic{},
)
