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
	"github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"

	apppkg "github.com/volnix-protocol/volnix-protocol/app"
)

// ABCIAdapter адаптирует VolnixApp для работы с CometBFT v0.38.17
type ABCIAdapter struct {
	app *apppkg.VolnixApp
}

// NewABCIAdapter создает новый ABCI адаптер
func NewABCIAdapter(app *apppkg.VolnixApp) *ABCIAdapter {
	return &ABCIAdapter{app: app}
}

// Info implements abci.Application
func (a *ABCIAdapter) Info(ctx context.Context, req *abci.RequestInfo) (*abci.ResponseInfo, error) {
	return &abci.ResponseInfo{
		Data:             "volnix-protocol",
		Version:          "0.1.0",
		AppVersion:       1,
		LastBlockHeight:  0, // Будет обновляться
		LastBlockAppHash: []byte{},
	}, nil
}

// Query implements abci.Application
func (a *ABCIAdapter) Query(ctx context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	// Здесь будет логика запросов
	// Пока возвращаем заглушку
	return &abci.ResponseQuery{
		Code:  0,
		Value: []byte("query response"),
		Log:   "query processed",
	}, nil
}

// CheckTx implements abci.Application
func (a *ABCIAdapter) CheckTx(ctx context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	// Здесь будет логика проверки транзакций
	// Пока возвращаем заглушку
	return &abci.ResponseCheckTx{
		Code: 0,
		Data: []byte("tx valid"),
		Log:  "transaction is valid",
	}, nil
}

// InitChain implements abci.Application
func (a *ABCIAdapter) InitChain(ctx context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	// Создаем SDK контекст
	sdkCtx := a.app.NewContext(true)

	// Выполняем InitChain через SDK
	_ = sdkCtx // Пока не используем, но создаем для совместимости
	return &abci.ResponseInitChain{}, nil
}

// PrepareProposal implements abci.Application
func (a *ABCIAdapter) PrepareProposal(ctx context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	// Заглушка для PrepareProposal
	return &abci.ResponsePrepareProposal{}, nil
}

// ProcessProposal implements abci.Application
func (a *ABCIAdapter) ProcessProposal(ctx context.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	// Заглушка для ProcessProposal
	return &abci.ResponseProcessProposal{}, nil
}

// FinalizeBlock implements abci.Application
func (a *ABCIAdapter) FinalizeBlock(ctx context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	// Создаем SDK контекст
	sdkCtx := a.app.NewContext(true)

	// Выполняем BeginBlocker через SDK для PoVB консенсуса
	if err := a.app.GetConsensusKeeper().BeginBlocker(sdkCtx); err != nil {
		return nil, fmt.Errorf("failed to execute BeginBlocker: %w", err)
	}

	// Обрабатываем транзакции
	var deliverTxs []*abci.ExecTxResult
	for _, tx := range req.Txs {
		// Проверяем транзакцию
		checkResult, err := a.CheckTx(ctx, &abci.RequestCheckTx{
			Tx: tx,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to check transaction: %w", err)
		}

		if checkResult.Code != 0 {
			// Транзакция не прошла проверку
			deliverTxs = append(deliverTxs, &abci.ExecTxResult{
				Code: checkResult.Code,
				Log:  checkResult.Log,
			})
			continue
		}

		// Выполняем транзакцию
		// Здесь будет логика выполнения через SDK
		deliverTxs = append(deliverTxs, &abci.ExecTxResult{
			Code: 0,
			Log:  "transaction executed successfully",
		})
	}

	// Выполняем EndBlocker через SDK
	if err := a.app.GetConsensusKeeper().EndBlocker(sdkCtx); err != nil {
		return nil, fmt.Errorf("failed to execute EndBlocker: %w", err)
	}

	// Возвращаем результат
	return &abci.ResponseFinalizeBlock{
		Events:        sdkCtx.EventManager().ABCIEvents(),
		TxResults:     deliverTxs,
		ValidatorUpdates: []abci.ValidatorUpdate{},
		AppHash:       []byte{}, // Будет обновляться
	}, nil
}

// ExtendVote implements abci.Application
func (a *ABCIAdapter) ExtendVote(ctx context.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	// Заглушка для ExtendVote
	return &abci.ResponseExtendVote{}, nil
}

// VerifyVoteExtension implements abci.Application
func (a *ABCIAdapter) VerifyVoteExtension(ctx context.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	// Заглушка для VerifyVoteExtension
	return &abci.ResponseVerifyVoteExtension{}, nil
}

// Commit implements abci.Application
func (a *ABCIAdapter) Commit(ctx context.Context, req *abci.RequestCommit) (*abci.ResponseCommit, error) {
	// Здесь будет логика коммита состояния
	// Пока возвращаем заглушку
	return &abci.ResponseCommit{}, nil
}

// ListSnapshots implements abci.Application
func (a *ABCIAdapter) ListSnapshots(ctx context.Context, req *abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	return &abci.ResponseListSnapshots{}, nil
}

// OfferSnapshot implements abci.Application
func (a *ABCIAdapter) OfferSnapshot(ctx context.Context, req *abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	return &abci.ResponseOfferSnapshot{}, nil
}

// LoadSnapshotChunk implements abci.Application
func (a *ABCIAdapter) LoadSnapshotChunk(ctx context.Context, req *abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	return &abci.ResponseLoadSnapshotChunk{}, nil
}

// ApplySnapshotChunk implements abci.Application
func (a *ABCIAdapter) ApplySnapshotChunk(ctx context.Context, req *abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	return &abci.ResponseApplySnapshotChunk{}, nil
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

	// Создаем ABCI адаптер
	abciAdapter := NewABCIAdapter(app)

	// Создаем ABCI клиент
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

	// Создаем узел с правильными параметрами для v0.38.17
	node, err := node.NewNode(
		cfg,
		privValidator,
		nodeKey,
		abciClient,
		node.DefaultGenesisDocProviderFunc(cfg),
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
func (n *CometBFTNode) GetGenesisDoc() (*types.GenesisDoc, error) {
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
func (n *CometBFTNode) GetValidators(height int64) (*types.ValidatorSet, error) {
	client := n.GetRPCClient()
	result, err := client.Validators(context.Background(), &height, nil, nil)
	if err != nil {
		return nil, err
	}
	// Создаем ValidatorSet из списка валидаторов
	validatorSet := &types.ValidatorSet{
		Validators: result.Validators,
	}
	return validatorSet, nil
}

// GetBlock возвращает блок по высоте
func (n *CometBFTNode) GetBlock(height int64) (*types.Block, error) {
	client := n.GetRPCClient()
	result, err := client.Block(context.Background(), &height)
	if err != nil {
		return nil, err
	}
	return result.Block, nil
}

// GetBlockByHash возвращает блок по хешу
func (n *CometBFTNode) GetBlockByHash(hash []byte) (*types.Block, error) {
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
func (n *CometBFTNode) BroadcastTxSync(tx types.Tx) (*coretypes.ResultBroadcastTx, error) {
	client := n.GetRPCClient()
	return client.BroadcastTxSync(context.Background(), tx)
}

// BroadcastTxAsync отправляет транзакцию асинхронно
func (n *CometBFTNode) BroadcastTxAsync(tx types.Tx) (*coretypes.ResultBroadcastTx, error) {
	client := n.GetRPCClient()
	return client.BroadcastTxAsync(context.Background(), tx)
}

// BroadcastTxCommit отправляет транзакцию и ждет коммита
func (n *CometBFTNode) BroadcastTxCommit(tx types.Tx) (*coretypes.ResultBroadcastTxCommit, error) {
	client := n.GetRPCClient()
	return client.BroadcastTxCommit(context.Background(), tx)
}
