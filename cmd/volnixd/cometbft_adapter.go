package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	sdklog "cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"

	apppkg "github.com/volnix-protocol/volnix-protocol/app"
)

// ABCIAdapter bridges CometBFT's context-aware ABCI to BaseApp
// by forwarding requests to BaseApp's methods.
type ABCIAdapter struct {
	app *apppkg.VolnixApp
}

func NewABCIAdapter(app *apppkg.VolnixApp) *ABCIAdapter { return &ABCIAdapter{app: app} }

func (a *ABCIAdapter) Info(ctx context.Context, req *abci.RequestInfo) (*abci.ResponseInfo, error) {
	res, err := a.app.GetBaseApp().Info(req)
	return res, err
}

func (a *ABCIAdapter) Query(ctx context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	res, err := a.app.GetBaseApp().Query(ctx, req)
	return res, err
}

func (a *ABCIAdapter) CheckTx(ctx context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	res, err := a.app.GetBaseApp().CheckTx(req)
	return res, err
}

func (a *ABCIAdapter) InitChain(ctx context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	res, err := a.app.GetBaseApp().InitChain(req)
	return res, err
}

func (a *ABCIAdapter) PrepareProposal(ctx context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	res, err := a.app.GetBaseApp().PrepareProposal(req)
	return res, err
}

func (a *ABCIAdapter) ProcessProposal(ctx context.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	res, err := a.app.GetBaseApp().ProcessProposal(req)
	return res, err
}

func (a *ABCIAdapter) FinalizeBlock(ctx context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	res, err := a.app.GetBaseApp().FinalizeBlock(req)
	return res, err
}

func (a *ABCIAdapter) ExtendVote(ctx context.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	res, err := a.app.GetBaseApp().ExtendVote(ctx, req)
	return res, err
}

func (a *ABCIAdapter) VerifyVoteExtension(ctx context.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	res, err := a.app.GetBaseApp().VerifyVoteExtension(req)
	return res, err
}

func (a *ABCIAdapter) Commit(ctx context.Context, req *abci.RequestCommit) (*abci.ResponseCommit, error) {
	res, err := a.app.GetBaseApp().Commit()
	return res, err
}

func (a *ABCIAdapter) ListSnapshots(ctx context.Context, req *abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	res, err := a.app.GetBaseApp().ListSnapshots(req)
	return res, err
}

func (a *ABCIAdapter) OfferSnapshot(ctx context.Context, req *abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	res, err := a.app.GetBaseApp().OfferSnapshot(req)
	return res, err
}

func (a *ABCIAdapter) LoadSnapshotChunk(ctx context.Context, req *abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	res, err := a.app.GetBaseApp().LoadSnapshotChunk(req)
	return res, err
}

func (a *ABCIAdapter) ApplySnapshotChunk(ctx context.Context, req *abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	res, err := a.app.GetBaseApp().ApplySnapshotChunk(req)
	return res, err
}

// CometBFTNode представляет полноценный блокчейн узел
type CometBFTNode struct {
	node    *node.Node
	app     *apppkg.VolnixApp
	homeDir string
	logger  log.Logger
}

// NewCometBFTNode создает новый CometBFT узел
func NewCometBFTNode(homeDir string, logger log.Logger) (*CometBFTNode, error) {
	// Создаем базу данных для приложения
	dbPath := filepath.Join(homeDir, "data")
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	database, err := dbm.NewDB("cometbft_app", dbm.GoLevelDBBackend, dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Создаем приложение с совместимым логгером
	encoding := apppkg.MakeEncodingConfig()
	sdkLogger := sdklog.NewNopLogger() // Используем SDK логгер для приложения
	app := apppkg.NewVolnixApp(sdkLogger, database, nil, encoding)

	// Загружаем последнюю версию
	if err := app.LoadLatestVersion(); err != nil {
		return nil, fmt.Errorf("failed to load latest version: %w", err)
	}

	// Создаем ABCI адаптер и клиент
	abciAdapter := NewABCIAdapter(app)
	abciClient := proxy.NewLocalClientCreator(abciAdapter)

	// Создаем конфигурацию CometBFT по умолчанию
	cfg := config.DefaultConfig()

	// Настраиваем пути
	cfg.SetRoot(homeDir)
	cfg.P2P.AddrBookStrict = false
	cfg.P2P.AllowDuplicateIP = true

	// Создаем приватный ключ валидатора
	privKeyFile := filepath.Join(homeDir, "config", "priv_validator_key.json")
	stateFile := filepath.Join(homeDir, "config", "priv_validator_state.json")

	privValidator := privval.LoadOrGenFilePV(privKeyFile, stateFile)

	// Создаем NodeKey для P2P
	nodeKey := &p2p.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	// Явный провайдер genesis.json
	genesisPath := filepath.Join(homeDir, "config", "genesis.json")
	genesisProvider := func() (*cmttypes.GenesisDoc, error) {
		return cmttypes.GenesisDocFromFile(genesisPath)
	}

	// Создаем узел с правильными параметрами для v0.38.17
	node, err := node.NewNode(
		cfg,
		privValidator,
		nodeKey,
		abciClient,
		genesisProvider,
		config.DefaultDBProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	return &CometBFTNode{
		node:    node,
		app:     app,
		homeDir: homeDir,
		logger:  logger,
	}, nil
}

// Start запускает CometBFT узел
func (n *CometBFTNode) Start() error {
	// Запускаем узел
	if err := n.node.Start(); err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}

	n.logger.Info("🚀 CometBFT node started successfully!")
	n.logger.Info("📡 Chain ID: test-volnix")
	n.logger.Info(fmt.Sprintf("🌐 RPC: http://%s", n.node.Config().RPC.ListenAddress))
	n.logger.Info(fmt.Sprintf("🔗 P2P: %s", n.node.Config().P2P.ListenAddress))
	n.logger.Info(fmt.Sprintf("📊 Database: %s", filepath.Join(n.homeDir, "data")))
	n.logger.Info("💾 Storage: Persistent (GoLevelDB)")
	n.logger.Info("✅ Full blockchain node is running...")

	return nil
}

// Stop останавливает CometBFT узел
func (n *CometBFTNode) Stop() error {
	if err := n.node.Stop(); err != nil {
		return fmt.Errorf("failed to stop node: %w", err)
	}
	return nil
}

// WaitForShutdown ждет сигнала завершения
func (n *CometBFTNode) WaitForShutdown() {
	// Ждем сигнала завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	n.logger.Info("🛑 Shutting down CometBFT node...")
	if err := n.Stop(); err != nil {
		n.logger.Error("Failed to stop node", "error", err)
	}
}

// GetRPCClient возвращает RPC клиент для внешних запросов
func (n *CometBFTNode) GetRPCClient() *local.Local {
	return local.New(n.node)
}

// GetApp возвращает приложение для прямого доступа
func (n *CometBFTNode) GetApp() *apppkg.VolnixApp {
	return n.app
}

// IsRunning проверяет, запущен ли узел
func (n *CometBFTNode) IsRunning() bool {
	return n.node.IsRunning()
}

// GetNodeInfo возвращает информацию об узле
func (n *CometBFTNode) GetNodeInfo() (*p2p.DefaultNodeInfo, error) {
	nodeInfo := n.node.NodeInfo()
	if defaultNodeInfo, ok := nodeInfo.(*p2p.DefaultNodeInfo); ok {
		return defaultNodeInfo, nil
	}
	return nil, fmt.Errorf("failed to get node info")
}

// GetGenesisDoc возвращает genesis документ
func (n *CometBFTNode) GetGenesisDoc() (*cmttypes.GenesisDoc, error) {
	genesisDoc := n.node.GenesisDoc()
	if genesisDoc == nil {
		return nil, fmt.Errorf("genesis doc not available")
	}
	return genesisDoc, nil
}

// GetConsensusState возвращает состояние консенсуса
func (n *CometBFTNode) GetConsensusState() (*abci.ResponseQuery, error) {
	client := n.GetRPCClient()
	result, err := client.ABCIQuery(context.Background(), "/consensus/state", nil)
	if err != nil {
		return nil, err
	}

	// Конвертируем результат в ResponseQuery
	response := &abci.ResponseQuery{
		Code:      result.Response.Code,
		Log:       result.Response.Log,
		Info:      result.Response.Info,
		Index:     result.Response.Index,
		Key:       result.Response.Key,
		Value:     result.Response.Value,
		ProofOps:  result.Response.ProofOps,
		Height:    result.Response.Height,
		Codespace: result.Response.Codespace,
	}

	return response, nil
}

// GetBlockHeight возвращает текущую высоту блока
func (n *CometBFTNode) GetBlockHeight() (int64, error) {
	client := n.GetRPCClient()
	status, err := client.Status(context.Background())
	if err != nil {
		return 0, err
	}
	return status.SyncInfo.LatestBlockHeight, nil
}

// GetValidators возвращает список валидаторов
func (n *CometBFTNode) GetValidators(height int64) (*cmttypes.ValidatorSet, error) {
	client := n.GetRPCClient()
	result, err := client.Validators(context.Background(), &height, nil, nil)
	if err != nil {
		return nil, err
	}
	// Создаем ValidatorSet из списка валидаторов
	validatorSet := &cmttypes.ValidatorSet{
		Validators: result.Validators,
	}
	return validatorSet, nil
}

// GetBlock возвращает блок по высоте
func (n *CometBFTNode) GetBlock(height int64) (*cmttypes.Block, error) {
	client := n.GetRPCClient()
	result, err := client.Block(context.Background(), &height)
	if err != nil {
		return nil, err
	}
	return result.Block, nil
}

// GetBlockByHash возвращает блок по хешу
func (n *CometBFTNode) GetBlockByHash(hash []byte) (*cmttypes.Block, error) {
	client := n.GetRPCClient()
	result, err := client.BlockByHash(context.Background(), hash)
	if err != nil {
		return nil, err
	}
	return result.Block, nil
}

// GetTx возвращает транзакцию по хешу
func (n *CometBFTNode) GetTx(hash []byte) (*abci.TxResult, error) {
	client := n.GetRPCClient()
	result, err := client.Tx(context.Background(), hash, false)
	if err != nil {
		return nil, err
	}
	// Конвертируем результат в TxResult
	txResult := &abci.TxResult{
		Height: result.Height,
		Index:  result.Index,
		Tx:     result.Tx,
		Result: result.TxResult,
	}
	return txResult, nil
}

// BroadcastTxSync отправляет транзакцию синхронно
func (n *CometBFTNode) BroadcastTxSync(tx cmttypes.Tx) (*coretypes.ResultBroadcastTx, error) {
	client := n.GetRPCClient()
	return client.BroadcastTxSync(context.Background(), tx)
}

// BroadcastTxAsync отправляет транзакцию асинхронно
func (n *CometBFTNode) BroadcastTxAsync(tx cmttypes.Tx) (*coretypes.ResultBroadcastTx, error) {
	client := n.GetRPCClient()
	return client.BroadcastTxAsync(context.Background(), tx)
}

// BroadcastTxCommit отправляет транзакцию и ждет коммита
func (n *CometBFTNode) BroadcastTxCommit(tx cmttypes.Tx) (*coretypes.ResultBroadcastTxCommit, error) {
	client := n.GetRPCClient()
	return client.BroadcastTxCommit(context.Background(), tx)
}
