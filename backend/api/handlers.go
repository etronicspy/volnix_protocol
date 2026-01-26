package main

import (
	"encoding/json"
	"net/http"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
)

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
	
	// Placeholder: Query method not yet defined in proto
	// Return empty response for now
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orders": []interface{}{},
		"message": "Orders query endpoint - coming soon",
	})
}

func (s *Server) anteilAuctionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}
	
	s.setCORSHeaders(w)
	
	// Placeholder: Query method not yet defined in proto
	// Return empty response for now
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"auctions": []interface{}{},
		"message": "Auctions query endpoint - coming soon",
	})
}
