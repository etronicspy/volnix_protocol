package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TxTransaction represents a blockchain transaction for the API response
type TxTransaction struct {
	Hash      string `json:"hash"`
	Height    int64  `json:"height"`
	Timestamp string `json:"timestamp"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    string `json:"amount"`
	Denom     string `json:"denom"`
	Status    string `json:"status"`
	TxType    string `json:"tx_type,omitempty"` // "transfer" | "identity_verified" | "role_changed"
}

// txCacheEntry holds cached transactions with expiry
type txCacheEntry struct {
	txs       []TxTransaction
	height    int64
	expiresAt time.Time
}

var txCache = struct {
	sync.RWMutex
	entries map[string]*txCacheEntry
}{entries: make(map[string]*txCacheEntry)}

const txCacheTTL = 15 * time.Second

func (s *Server) txTransactionsHandler(w http.ResponseWriter, r *http.Request) {
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

	// Extract address from path: /volnix/tx/v1/transactions/{address}
	prefix := "/volnix/tx/v1/transactions/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	address := strings.TrimPrefix(r.URL.Path, prefix)
	if address == "" {
		http.Error(w, "Address is required", http.StatusBadRequest)
		return
	}

	// Optional limit query param
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	txs, err := s.scanTransactionsForAddress(address, limit)
	if err != nil {
		log.Printf("tx scan error for %s: %v", address, err)
		http.Error(w, fmt.Sprintf("Failed to scan transactions: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transactions": txs,
	})
}

const maxBlockScanDepth = 2000 // Limit scan depth to avoid timeout with many blocks

func (s *Server) scanTransactionsForAddress(address string, limit int) ([]TxTransaction, error) {
	// Check cache
	txCache.RLock()
	if e, ok := txCache.entries[address]; ok && time.Now().Before(e.expiresAt) {
		txs := e.txs
		txCache.RUnlock()
		if len(txs) > limit {
			txs = txs[:limit]
		}
		return txs, nil
	}
	txCache.RUnlock()

	// 1. Try tx_search (fast, indexed) - paginate and filter by address
	if txs, err := s.txSearchByAddress(address, limit); err == nil && len(txs) > 0 {
		s.cacheTxResult(address, txs)
		if len(txs) > limit {
			txs = txs[:limit]
		}
		return txs, nil
	}

	// 2. Fallback: block scan with limited depth
	latest, err := s.getLatestBlockHeight()
	if err != nil {
		return nil, err
	}
	if latest == 0 {
		return []TxTransaction{}, nil
	}

	var result []TxTransaction
	seen := make(map[string]bool)
	scanned := 0

	for h := latest; h >= 1 && scanned < maxBlockScanDepth; h-- {
		scanned++
		txs, err := s.scanBlockForAddress(h, address, seen)
		if err != nil {
			log.Printf("scan block %d: %v", h, err)
			continue
		}
		for _, tx := range txs {
			if !seen[tx.Hash] {
				seen[tx.Hash] = true
				result = append(result, tx)
			}
		}
		if len(result) >= limit {
			break
		}
	}

	s.cacheTxResult(address, result)

	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *Server) cacheTxResult(address string, txs []TxTransaction) {
	latest, _ := s.getLatestBlockHeight()
	txCache.Lock()
	txCache.entries[address] = &txCacheEntry{
		txs:       txs,
		height:    latest,
		expiresAt: time.Now().Add(txCacheTTL),
	}
	txCache.Unlock()
}

// txSearchByAddress uses tx_search RPC and filters by address (fast when indexer is on)
func (s *Server) txSearchByAddress(address string, limit int) ([]TxTransaction, error) {
	var all []TxTransaction
	page := 1
	maxPages := 20 // cap at 2000 txs

	for page <= maxPages {
		reqBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tx_search",
			"params": map[string]interface{}{
				"query":    "tx.height>=1",
				"prove":    false,
				"page":     strconv.Itoa(page),
				"per_page": "100",
				"order_by": "desc",
			},
		}
		body, _ := json.Marshal(reqBody)
		resp, err := http.Post(s.rpcEndpoint, "application/json", strings.NewReader(string(body)))
		if err != nil {
			return nil, err
		}
		var data struct {
			Result struct {
				Txs []struct {
					Hash     string `json:"hash"`
					Height   string `json:"height"`
					TxResult struct {
						Code   int `json:"code"`
						Events []struct {
							Type       string `json:"type"`
							Attributes []struct {
								Key   string `json:"key"`
								Value string `json:"value"`
								Index bool   `json:"index"`
							} `json:"attributes"`
						} `json:"events"`
					} `json:"tx_result"`
				} `json:"txs"`
				TotalCount string `json:"total_count"`
			} `json:"result"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		if data.Error != nil {
			return nil, fmt.Errorf("tx_search: %s", data.Error.Message)
		}
		for _, item := range data.Result.Txs {
			tx := s.parseTxSearchItem(item, address)
			if tx != nil {
				all = append(all, *tx)
				if len(all) >= limit {
					return all, nil
				}
			}
		}
		if len(data.Result.Txs) < 100 {
			break
		}
		page++
	}
	return all, nil
}

func (s *Server) parseTxSearchItem(item struct {
	Hash     string `json:"hash"`
	Height   string `json:"height"`
	TxResult struct {
		Code   int `json:"code"`
		Events []struct {
			Type       string `json:"type"`
			Attributes []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
				Index bool   `json:"index"`
			} `json:"attributes"`
		} `json:"events"`
	} `json:"tx_result"`
}, address string) *TxTransaction {
	var from, to, amount, denom string
	var txType string
	for _, ev := range item.TxResult.Events {
		if ev.Type == "transfer" || ev.Type == "coin_spent" || ev.Type == "coin_received" {
			for _, attr := range ev.Attributes {
				k, v := attr.Key, attr.Value
				if !attr.Index {
					k = decodeBase64(k)
					v = decodeBase64(v)
				}
				switch k {
				case "sender", "spender":
					from = v
				case "recipient", "receiver":
					to = v
				case "amount":
					if amt, d := parseAmount(v); amt != "" {
						amount, denom = amt, d
					}
				}
			}
			continue
		}
		if ev.Type == "ident.identity_verified" || ev.Type == "ident.role_changed" {
			var addr string
			for _, a := range ev.Attributes {
				k := a.Key
				if !a.Index {
					k = decodeBase64(k)
				}
				if k == "address" {
					addr = a.Value
					if !a.Index {
						addr = decodeBase64(addr)
					}
					break
				}
			}
			if addr == address {
				txType = "identity_verified"
				if ev.Type == "ident.role_changed" {
					txType = "role_changed"
				}
				from = addr
				to = ""
				amount = "0"
				denom = "identity"
				if txType == "role_changed" {
					denom = "role"
				}
			}
		}
	}
	relevant := (from == address || to == address) || txType != ""
	if !relevant {
		return nil
	}
	status := "failed"
	if item.TxResult.Code == 0 {
		status = "success"
	}
	if txType == "" {
		txType = "transfer"
	}
	h, _ := strconv.ParseInt(item.Height, 10, 64)
	return &TxTransaction{
		Hash:      item.Hash,
		Height:    h,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		From:      or(from, address),
		To:        or(to, address),
		Amount:    or(amount, "0"),
		Denom:     or(denom, "uwrt"),
		Status:    status,
		TxType:    txType,
	}
}

func (s *Server) getLatestBlockHeight() (int64, error) {
	resp, err := http.Get(s.rpcEndpoint + "/status")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data struct {
		Result struct {
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
			} `json:"sync_info"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	h := data.Result.SyncInfo.LatestBlockHeight
	if h == "" {
		return 0, nil
	}
	return strconv.ParseInt(h, 10, 64)
}

func (s *Server) scanBlockForAddress(height int64, address string, seen map[string]bool) ([]TxTransaction, error) {
	blockURL := fmt.Sprintf("%s/block?height=%d", s.rpcEndpoint, height)
	resultsURL := fmt.Sprintf("%s/block_results?height=%d", s.rpcEndpoint, height)

	blockResp, err := http.Get(blockURL)
	if err != nil {
		return nil, err
	}
	defer blockResp.Body.Close()

	resultsResp, err := http.Get(resultsURL)
	if err != nil {
		return nil, err
	}
	defer resultsResp.Body.Close()

	var blockData struct {
		Result struct {
			Block struct {
				Data struct {
					Txs []string `json:"txs"`
				} `json:"data"`
			} `json:"block"`
		} `json:"result"`
	}
	if err := json.NewDecoder(blockResp.Body).Decode(&blockData); err != nil {
		return nil, err
	}

	var resultsData struct {
		Result struct {
			TxsResults []struct {
				Code   int `json:"code"`
				Events []struct {
					Type       string `json:"type"`
					Attributes []struct {
						Key   string `json:"key"`
						Value string `json:"value"`
						Index bool   `json:"index"`
					} `json:"attributes"`
				} `json:"events"`
			} `json:"txs_results"`
		} `json:"result"`
	}
	body, _ := io.ReadAll(resultsResp.Body)
	if err := json.Unmarshal(body, &resultsData); err != nil {
		return nil, err
	}

	txs := blockData.Result.Block.Data.Txs
	txResults := resultsData.Result.TxsResults
	var result []TxTransaction

	for i := 0; i < len(txResults) && i < len(txs); i++ {
		events := txResults[i].Events
		var from, to, amount, denom string
		var txType string

		for _, ev := range events {
			// Bank/transfer events
			if ev.Type == "transfer" || ev.Type == "coin_spent" || ev.Type == "coin_received" {
				for _, attr := range ev.Attributes {
					k, v := attr.Key, attr.Value
					if !attr.Index {
						k = decodeBase64(k)
						v = decodeBase64(v)
					}
					switch k {
					case "sender", "spender":
						from = v
					case "recipient", "receiver":
						to = v
					case "amount":
						if amt, d := parseAmount(v); amt != "" {
							amount, denom = amt, d
						}
					}
				}
				continue
			}
			// Ident events: identity_verified, role_changed
			if ev.Type == "ident.identity_verified" || ev.Type == "ident.role_changed" {
				addr := getAttr(ev.Attributes, "address")
				if addr == address {
					txType = "identity_verified"
					if ev.Type == "ident.role_changed" {
						txType = "role_changed"
					}
					from = addr
					to = ""
					amount = "0"
					denom = "identity"
					if txType == "role_changed" {
						denom = "role"
					}
				}
			}
		}

		// Transfer: relevant if from or to matches
		relevant := (from == address || to == address)
		// Ident: relevant if we found matching ident event
		if txType != "" {
			relevant = true
		}
		if !relevant {
			continue
		}

		txHash := txHashFromBase64(txs[i])
		if txHash == "" || seen[txHash] {
			continue
		}

		status := "failed"
		if txResults[i].Code == 0 {
			status = "success"
		}

		if txType == "" {
			txType = "transfer"
		}

		result = append(result, TxTransaction{
			Hash:      txHash,
			Height:    height,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			From:      or(from, address),
			To:        or(to, address),
			Amount:    or(amount, "0"),
			Denom:     or(denom, "uwrt"),
			Status:    status,
			TxType:    txType,
		})
	}
	return result, nil
}

func decodeBase64(s string) string {
	if s == "" {
		return ""
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s
	}
	return string(b)
}

func getAttr(attrs []struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Index bool   `json:"index"`
}, key string) string {
	for _, a := range attrs {
		k := a.Key
		if !a.Index {
			k = decodeBase64(k)
		}
		if k == key {
			v := a.Value
			if !a.Index {
				v = decodeBase64(v)
			}
			return v
		}
	}
	return ""
}

var amountRegex = regexp.MustCompile(`^(\d+)(\w+)$`)

func parseAmount(s string) (amount, denom string) {
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if m := amountRegex.FindStringSubmatch(part); len(m) == 3 {
			return m[1], m[2]
		}
	}
	return "", ""
}

func txHashFromBase64(txBase64 string) string {
	if txBase64 == "" {
		return ""
	}
	b, err := base64.StdEncoding.DecodeString(txBase64)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return strings.ToUpper(hex.EncodeToString(h[:]))
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
