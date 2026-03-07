package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cosmossdk.io/log"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
)

// MonitoringService provides monitoring and metrics for Volnix Protocol
type MonitoringService struct {
	app       *VolnixApp
	logger    log.Logger
	server    *http.Server
	startedAt time.Time
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(app *VolnixApp, logger log.Logger) *MonitoringService {
	return &MonitoringService{
		app:       app,
		logger:    logger,
		startedAt: time.Now(),
	}
}

// Start starts the monitoring service
func (ms *MonitoringService) Start(port string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", ms.healthHandler)
	mux.HandleFunc("/metrics", ms.metricsHandler)
	mux.HandleFunc("/status", ms.statusHandler)
	mux.HandleFunc("/consensus", ms.consensusHandler)
	mux.HandleFunc("/economic", ms.economicHandler)
	mux.HandleFunc("/identity", ms.identityHandler)

	ms.server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	ms.logger.Info("starting monitoring service", "module", "monitoring", "port", port)
	return ms.server.ListenAndServe()
}

// Stop stops the monitoring service
func (ms *MonitoringService) Stop() error {
	if ms.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return ms.server.Shutdown(ctx)
	}
	return nil
}

// healthHandler provides health check endpoint
func (ms *MonitoringService) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "0.1.0-alpha",
		"chain_id":  "volnix-1",
	}

	if err := json.NewEncoder(w).Encode(health); err != nil {
		ms.logger.Error("failed to encode health response", "module", "monitoring", "error", err)
	}
}

// metricsHandler provides Prometheus-style metrics
func (ms *MonitoringService) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	metrics := ms.collectAllMetrics()

	for key, value := range metrics {
		fmt.Fprintf(w, "volnix_%s %v\n", key, value)
	}
}

// statusHandler provides overall system status
func (ms *MonitoringService) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := ms.app.NewContext(true)

	status := map[string]interface{}{
		"chain_id":      "volnix-1",
		"latest_height": ctx.BlockHeight(),
		"timestamp":     time.Now().Unix(),
		"modules": map[string]interface{}{
			"ident":     "active",
			"lizenz":    "active",
			"anteil":    "active",
			"consensus": "active",
		},
		"network": map[string]interface{}{
			"peers":      0,
			"validators": ms.getValidatorCount(),
			"consensus":  "PoVB",
		},
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		ms.logger.Error("failed to encode status response", "module", "monitoring", "error", err)
	}
}

// consensusHandler provides consensus-specific metrics
func (ms *MonitoringService) consensusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ms.getConsensusMetrics()); err != nil {
		ms.logger.Error("failed to encode consensus response", "module", "monitoring", "error", err)
	}
}

// economicHandler provides economic metrics
func (ms *MonitoringService) economicHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ms.getEconomicMetrics()); err != nil {
		ms.logger.Error("failed to encode economic response", "module", "monitoring", "error", err)
	}
}

// identityHandler provides identity system metrics
func (ms *MonitoringService) identityHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ms.getIdentityMetrics()); err != nil {
		ms.logger.Error("failed to encode identity response", "module", "monitoring", "error", err)
	}
}

// collectAllMetrics collects metrics from all modules
func (ms *MonitoringService) collectAllMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	ctx := ms.app.NewContext(true)
	metrics["uptime_seconds"] = int64(time.Since(ms.startedAt).Seconds())
	metrics["chain_height"] = ctx.BlockHeight()

	consensusMetrics := ms.getConsensusMetrics()
	metrics["consensus_validators_total"] = consensusMetrics["total_validators"]
	metrics["consensus_validators_active"] = consensusMetrics["active_validators"]
	metrics["consensus_burned_tokens"] = consensusMetrics["total_burned_tokens"]

	economicMetrics := ms.getEconomicMetrics()
	metrics["economic_orders_total"] = economicMetrics["total_orders"]
	metrics["economic_orders_active"] = economicMetrics["active_orders"]
	metrics["economic_volume_24h"] = economicMetrics["volume_24h"]

	identityMetrics := ms.getIdentityMetrics()
	metrics["identity_verified_accounts"] = identityMetrics["verified_accounts"]
	metrics["identity_pending_verifications"] = identityMetrics["pending_verifications"]

	return metrics
}

// getConsensusMetrics gets consensus system metrics from keeper
func (ms *MonitoringService) getConsensusMetrics() map[string]interface{} {
	metrics := map[string]interface{}{
		"total_validators":    0,
		"active_validators":   0,
		"total_burned_tokens": 0,
		"total_weight":        0,
		"halving_count":       0,
		"next_halving_height": 0,
	}

	if ms.app.consensusKeeper == nil {
		return metrics
	}

	ctx := ms.app.NewContext(true)

	validators := ms.app.consensusKeeper.GetAllValidators(ctx)
	metrics["total_validators"] = len(validators)
	activeCount := 0
	for _, v := range validators {
		if v.Status == consensusv1.ValidatorStatus_VALIDATOR_STATUS_ACTIVE {
			activeCount++
		}
	}
	metrics["active_validators"] = activeCount

	// Total burned tokens from consensus state
	if state, err := ms.app.consensusKeeper.GetConsensusState(ctx); err == nil {
		metrics["total_burned_tokens"] = state.TotalAntBurned
	}

	// Halving info
	if halvingInfo, err := ms.app.consensusKeeper.GetHalvingInfo(ctx); err == nil {
		metrics["next_halving_height"] = halvingInfo.NextHalvingHeight
		if halvingInfo.HalvingInterval > 0 {
			metrics["halving_count"] = halvingInfo.LastHalvingHeight / halvingInfo.HalvingInterval
		}
	}

	return metrics
}

// getEconomicMetrics gets economic system metrics from keeper
func (ms *MonitoringService) getEconomicMetrics() map[string]interface{} {
	metrics := map[string]interface{}{
		"total_orders":     0,
		"active_orders":    0,
		"completed_orders": 0,
		"volume_24h":       0,
		"total_volume":     0,
		"active_auctions":  0,
		"avg_price":        0,
	}

	if ms.app.anteilKeeper == nil {
		return metrics
	}

	ctx := ms.app.NewContext(true)

	orders, err := ms.app.anteilKeeper.GetAllOrders(ctx)
	if err == nil {
		metrics["total_orders"] = len(orders)
		activeCount := 0
		for _, order := range orders {
			if order.Status == anteilv1.OrderStatus_ORDER_STATUS_OPEN {
				activeCount++
			}
		}
		metrics["active_orders"] = activeCount
	}

	auctions, err := ms.app.anteilKeeper.GetAllAuctions(ctx)
	if err == nil {
		activeAuctions := 0
		for _, auction := range auctions {
			if auction.Status == anteilv1.AuctionStatus_AUCTION_STATUS_OPEN {
				activeAuctions++
			}
		}
		metrics["active_auctions"] = activeAuctions
	}

	trades, err := ms.app.anteilKeeper.GetAllTrades(ctx)
	if err == nil {
		metrics["completed_orders"] = len(trades)

		// volume_24h: sum total_value for trades with executed_at in last 24h
		cutoff := ctx.BlockTime().Add(-24 * time.Hour)
		var vol24h int64
		for _, t := range trades {
			if t.ExecutedAt != nil && t.ExecutedAt.AsTime().After(cutoff) {
				var v int64
				if _, err := fmt.Sscanf(t.TotalValue, "%d", &v); err == nil {
					vol24h += v
				}
			}
		}
		metrics["volume_24h"] = vol24h
	}

	// Total volume and average price from economic engine
	engine := anteilkeeper.NewEconomicEngine(ms.app.anteilKeeper)
	if marketMetrics, err := engine.CalculateMarketMetrics(ctx); err == nil {
		metrics["total_volume"] = marketMetrics.TotalVolume.String()
		metrics["avg_price"] = marketMetrics.AveragePrice.String()
	}

	return metrics
}

// getIdentityMetrics gets identity system metrics from keeper
func (ms *MonitoringService) getIdentityMetrics() map[string]interface{} {
	metrics := map[string]interface{}{
		"verified_accounts":         0,
		"pending_verifications":     0,
		"total_accounts":            0,
		"role_migrations":           0,
		"verification_success_rate": 0.0,
		"citizens":                  0,
		"validators":                0,
		"guests":                    0,
	}

	if ms.app.identKeeper == nil {
		return metrics
	}

	ctx := ms.app.NewContext(true)

	accounts, err := ms.app.identKeeper.GetAllVerifiedAccounts(ctx)
	if err == nil {
		metrics["verified_accounts"] = len(accounts)
		metrics["total_accounts"] = len(accounts)

		citizens := 0
		validators := 0
		guests := 0

		for _, account := range accounts {
			switch account.Role {
			case identv1.Role_ROLE_CITIZEN:
				citizens++
			case identv1.Role_ROLE_VALIDATOR:
				validators++
			case identv1.Role_ROLE_GUEST:
				guests++
			}
		}

		metrics["citizens"] = citizens
		metrics["validators"] = validators
		metrics["guests"] = guests
	}

	return metrics
}

// getValidatorCount gets the current validator count from keeper
func (ms *MonitoringService) getValidatorCount() int {
	if ms.app.consensusKeeper == nil {
		return 0
	}

	ctx := ms.app.NewContext(true)
	validators := ms.app.consensusKeeper.GetAllValidators(ctx)
	return len(validators)
}
