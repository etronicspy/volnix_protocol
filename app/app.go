package app

import (
	"io"

	sdklog "cosmossdk.io/log"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
)

// HelvetiaApp is a minimal placeholder app. It will be expanded in later iterations.
type HelvetiaApp struct {
	*baseapp.BaseApp
	appCodec codec.Codec
}

func NewHelvetiaApp(logger sdklog.Logger, db cosmosdb.DB, traceStore io.Writer, encoding EncodingConfig) *HelvetiaApp {
	bapp := baseapp.NewBaseApp("helvetia", logger, db, encoding.TxConfig.TxDecoder())
	bapp.SetVersion("0.1.0")
	return &HelvetiaApp{BaseApp: bapp, appCodec: encoding.Codec}
}
