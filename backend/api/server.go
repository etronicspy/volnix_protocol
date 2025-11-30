package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"io"
	"bytes"

	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server represents the REST API server
type Server struct {
	consensusClient consensusv1.QueryClient
	rpcEndpoint     string
}

// NewServer creates a new REST API server
func NewServer(consensusClient consensusv1.QueryClient) *Server {
	return &Server{
		consensusClient: consensusClient,
		rpcEndpoint:     "http://localhost:26657",
	}
}

// SetupRoutes sets up HTTP routes
func (s *Server) SetupRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/", s.rootHandler)

	// Consensus module endpoints
	mux.HandleFunc("/volnix/consensus/v1/params", s.consensusParamsHandler)
	mux.HandleFunc("/volnix/consensus/v1/validators", s.consensusValidatorsHandler)
}

// setCORSHeaders sets CORS headers for cross-origin requests
func (s *Server) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// healthHandler handles health check requests
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.setCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "volnix-rest-api",
	})
}

// rootHandler handles root requests
func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	s.setCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "Volnix REST API",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"health": "/health",
			"consensus_params": "/volnix/consensus/v1/params",
			"consensus_validators": "/volnix/consensus/v1/validators",
		},
	})
}

// consensusParamsHandler handles consensus params requests
func (s *Server) consensusParamsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.setCORSHeaders(w)
	
	// If gRPC is not available, return default params instead of error
	if s.consensusClient == nil {
		log.Printf("gRPC client not available, returning default params")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"params": map[string]interface{}{
				"base_block_time":           "5s",
				"high_activity_threshold":   "1000",
				"low_activity_threshold":     "100",
				"min_burn_amount":           "10",
				"max_burn_amount":           "1000",
			},
		})
		return
	}

	ctx := r.Context()
	resp, err := s.consensusClient.Params(ctx, &consensusv1.QueryParamsRequest{})
	if err != nil {
		log.Printf("Failed to get params from gRPC: %v, returning default params", err)
		// Return default params instead of error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"params": map[string]interface{}{
				"base_block_time":           "5s",
				"high_activity_threshold":   "1000",
				"low_activity_threshold":     "100",
				"min_burn_amount":           "10",
				"max_burn_amount":           "1000",
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// getValidatorsFromRPC gets validators from CometBFT RPC endpoint
func (s *Server) getValidatorsFromRPC() ([]map[string]interface{}, error) {
	// Query validators from RPC
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "validators",
		"params": map[string]interface{}{
			"height": 0, // 0 means latest
		},
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(s.rpcEndpoint, "application/json", bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp map[string]interface{}
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, err
	}

	result, ok := rpcResp["result"].(map[string]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	validators, ok := result["validators"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	// Convert to our format
	formattedValidators := make([]map[string]interface{}, 0, len(validators))
	for _, v := range validators {
		val, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		address, _ := val["address"].(string)
		votingPower, _ := val["voting_power"].(string)

		formattedValidators = append(formattedValidators, map[string]interface{}{
			"validator":          address,
			"status":             "VALIDATOR_STATUS_ACTIVE",
			"ant_balance":        "0",
			"activity_score":     "0",
			"total_blocks_created": 0,
			"total_burn_amount":  "0",
			"voting_power":       votingPower,
		})
	}

	return formattedValidators, nil
}

// consensusValidatorsHandler handles consensus validators requests
func (s *Server) consensusValidatorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		s.setCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.setCORSHeaders(w)
	
	// Try gRPC first
	if s.consensusClient != nil {
		ctx := r.Context()
		resp, err := s.consensusClient.Validators(ctx, &consensusv1.QueryValidatorsRequest{})
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		log.Printf("gRPC query failed: %v, trying RPC fallback", err)
	}

	// Fallback to RPC
	validators, err := s.getValidatorsFromRPC()
	if err != nil {
		log.Printf("RPC fallback failed: %v, returning empty list", err)
		validators = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"validators": validators,
	})
}

// handleError handles gRPC errors and converts them to HTTP errors
func (s *Server) handleError(w http.ResponseWriter, err error, message string) {
	st, ok := status.FromError(err)
	if !ok {
		log.Printf("%s: %v", message, err)
		http.Error(w, fmt.Sprintf("%s: %v", message, err), http.StatusInternalServerError)
		return
	}

	var httpStatus int
	switch st.Code() {
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable
	default:
		httpStatus = http.StatusInternalServerError
	}

	log.Printf("%s: %v (code: %s)", message, err, st.Code())
	http.Error(w, fmt.Sprintf("%s: %s", message, st.Message()), httpStatus)
}

