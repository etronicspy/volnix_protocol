package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cosmossdk.io/log"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
)

// MonitoringService provides monitoring and metrics for Volnix Protocol
type MonitoringService struct {
	app    *VolnixApp
	logger log.Logger
	server *http.Server
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(app *VolnixApp, logger log.Logger) *MonitoringService {
	return &MonitoringService{
		app:    app,
		logger: logger,
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

	ms.logger.Info("Starting monitoring service", "port", port)
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

	json.NewEncoder(w).Encode(health)
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

	json.NewEncoder(w).Encode(status)
}

// consensusHandler provides consensus-specific metrics
func (ms *MonitoringService) consensusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ms.getConsensusMetrics())
}

// economicHandler provides economic metrics
func (ms *MonitoringService) economicHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ms.getEconomicMetrics())
}

// identityHandler provides identity system metrics
func (ms *MonitoringService) identityHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ms.getIdentityMetrics())
}

// collectAllMetrics collects metrics from all modules
func (ms *MonitoringService) collectAllMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	ctx := ms.app.NewContext(true)
	metrics["uptime_seconds"] = time.Now().Unix()
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
		if v.Status == 1 { // ACTIVE
			activeCount++
		}
	}
	metrics["active_validators"] = activeCount

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
