package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
)

// ============================================================================
// Status Handler
// ============================================================================

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	modules := map[string]string{
		"ident":    "unknown",
		"lizenz":   "unknown",
		"anteil":   "unknown",
		"consensus": "unknown",
	}

	if s.identClient != nil {
		_, err := s.identClient.Params(r.Context(), &identv1.QueryParamsRequest{})
		if err == nil {
			modules["ident"] = "active"
		} else {
			modules["ident"] = "degraded"
		}
	}
	if s.lizenzClient != nil {
		_, err := s.lizenzClient.Params(r.Context(), &lizenzv1.QueryParamsRequest{})
		if err == nil {
			modules["lizenz"] = "active"
		} else {
			modules["lizenz"] = "degraded"
		}
	}
	if s.anteilClient != nil {
		_, err := s.anteilClient.Params(r.Context(), &anteilv1.QueryParamsRequest{})
		if err == nil {
			modules["anteil"] = "active"
		} else {
			modules["anteil"] = "degraded"
		}
	}
	if s.consensusClient != nil {
		_, err := s.consensusClient.Params(r.Context(), &consensusv1.QueryParamsRequest{})
		if err == nil {
			modules["consensus"] = "active"
		} else {
			modules["consensus"] = "degraded"
		}
	}

	latestHeight := int64(0)
	if resp, err := http.Get(s.rpcEndpoint + "/status"); err == nil {
		defer resp.Body.Close()
		var data struct {
			Result struct {
				SyncInfo struct {
					LatestBlockHeight string `json:"latest_block_height"`
				} `json:"sync_info"`
				NodeInfo struct {
					Network string `json:"network"`
				} `json:"node_info"`
			} `json:"result"`
		}
		if json.NewDecoder(resp.Body).Decode(&data) == nil {
			if h := data.Result.SyncInfo.LatestBlockHeight; h != "" {
				if v, e := strconv.ParseInt(h, 10, 64); e == nil {
					latestHeight = v
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"chain_id":      "volnix-1",
		"latest_height": latestHeight,
		"modules":       modules,
	})
}

// ============================================================================
// Identity Module Handlers
// ============================================================================

func (s *Server) identParamsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}
	
	s.setCORSHeaders(w)
	if s.identClient == nil {
		http.Error(w, "Identity service not available", http.StatusServiceUnavailable)
		return
	}
	
	ctx := r.Context()
	resp, err := s.identClient.Params(ctx, &identv1.QueryParamsRequest{})
	if err != nil {
		s.handleError(w, err, "Failed to get identity params")
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) identVerifiedAccountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}
	
	s.setCORSHeaders(w)
	if s.identClient == nil {
		http.Error(w, "Identity service not available", http.StatusServiceUnavailable)
		return
	}
	
	// Extract address from URL path
	address := r.URL.Path[len("/volnix/ident/v1/verified_account/"):]
	if address == "" {
		http.Error(w, "Address is required", http.StatusBadRequest)
		return
	}
	
	ctx := r.Context()
	resp, err := s.identClient.VerifiedAccount(ctx, &identv1.QueryVerifiedAccountRequest{
		Address: address,
	})
	if err != nil {
		s.handleError(w, err, "Failed to get verified account")
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) identVerifiedAccountsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}
	
	s.setCORSHeaders(w)
	if s.identClient == nil {
		http.Error(w, "Identity service not available", http.StatusServiceUnavailable)
		return
	}
	
	ctx := r.Context()
	resp, err := s.identClient.VerifiedAccounts(ctx, &identv1.QueryVerifiedAccountsRequest{})
	if err != nil {
		s.handleError(w, err, "Failed to get verified accounts")
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// Lizenz Module Handlers
// ============================================================================

func (s *Server) lizenzParamsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}
	
	s.setCORSHeaders(w)
	if s.lizenzClient == nil {
		http.Error(w, "Lizenz service not available", http.StatusServiceUnavailable)
		return
	}
	
	ctx := r.Context()
	resp, err := s.lizenzClient.Params(ctx, &lizenzv1.QueryParamsRequest{})
	if err != nil {
		s.handleError(w, err, "Failed to get lizenz params")
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) lizenzLizenzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	s.setCORSHeaders(w)
	if s.lizenzClient == nil {
		http.Error(w, "Lizenz service not available", http.StatusServiceUnavailable)
		return
	}

	// Extract validator from URL path
	validator := r.URL.Path[len("/volnix/lizenz/v1/lizenz/"):]
	if validator == "" {
		http.Error(w, "Validator address is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	resp, err := s.lizenzClient.ActivatedLizenz(ctx, &lizenzv1.QueryActivatedLizenzRequest{
		Validator: validator,
	})
	if err != nil {
		s.handleError(w, err, "Failed to get activated lizenz")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// Anteil Module Handlers
// ============================================================================

func (s *Server) anteilParamsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}
	
	s.setCORSHeaders(w)
	if s.anteilClient == nil {
		http.Error(w, "Anteil service not available", http.StatusServiceUnavailable)
		return
	}
	
	ctx := r.Context()
	resp, err := s.anteilClient.Params(ctx, &anteilv1.QueryParamsRequest{})
	if err != nil {
		s.handleError(w, err, "Failed to get anteil params")
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) anteilOrdersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}
	s.setCORSHeaders(w)
	if s.anteilClient == nil {
		http.Error(w, "Anteil service not available", http.StatusServiceUnavailable)
		return
	}
	ctx := r.Context()
	resp, err := s.anteilClient.Orders(ctx, &anteilv1.QueryOrdersRequest{})
	if err != nil {
		s.handleError(w, err, "Failed to get orders")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) anteilAuctionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}
	s.setCORSHeaders(w)
	if s.anteilClient == nil {
		http.Error(w, "Anteil service not available", http.StatusServiceUnavailable)
		return
	}
	ctx := r.Context()
	resp, err := s.anteilClient.Auctions(ctx, &anteilv1.QueryAuctionsRequest{})
	if err != nil {
		s.handleError(w, err, "Failed to get auctions")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
