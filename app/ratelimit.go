package app

import (
	"fmt"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/time/rate"
)

// RateLimiter provides optional rate limiting for CheckTx (RPC layer only).
// NOT used for consensus: mempool and fee market handle anti-spam. When enabled,
// use only for RPC protection (e.g. public node). Disabled by default.
type RateLimiter struct {
	globalLimiter   *rate.Limiter
	addressLimiters map[string]*addressEntry
	globalRate      rate.Limit
	perAddrRate     rate.Limit
	burstSize       int
	mu              sync.RWMutex
}

// addressEntry tracks a per-address limiter and its last access time.
type addressEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// GlobalRate is the global transaction rate limit (tx/sec)
	GlobalRate float64
	
	// PerAddrRate is the per-address transaction rate limit (tx/sec)
	PerAddrRate float64
	
	// BurstSize is the burst size for rate limiting
	BurstSize int
	
	// Enabled enables or disables rate limiting
	Enabled bool
}

// DefaultRateLimitConfig returns default rate limit configuration.
// Enabled: false — rate limiter does not affect consensus; enable only for RPC protection.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalRate:  1000.0, // 1000 tx/sec globally
		PerAddrRate: 10.0,   // 10 tx/sec per address
		BurstSize:   20,     // Allow bursts of 20 transactions
		Enabled:     false,  // Off by default; enable for RPC-only protection
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if !config.Enabled {
		return nil // Return nil if rate limiting is disabled
	}
	
	rl := &RateLimiter{
		globalLimiter:   rate.NewLimiter(rate.Limit(config.GlobalRate), config.BurstSize),
		addressLimiters: make(map[string]*addressEntry),
		globalRate:      rate.Limit(config.GlobalRate),
		perAddrRate:     rate.Limit(config.PerAddrRate),
		burstSize:       config.BurstSize,
	}
	
	return rl
}

// Allow checks if a transaction is allowed based on rate limiting
func (rl *RateLimiter) Allow(ctx sdk.Context, tx sdk.Tx) error {
	if rl == nil {
		return nil // Rate limiting disabled
	}
	
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	// Check global rate limit
	if !rl.globalLimiter.Allow() {
		return fmt.Errorf("global rate limit exceeded: %.0f tx/sec", float64(rl.globalRate))
	}
	
	// Check per-address rate limit
	// Get signers from transaction messages
	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		// Try to get signers from message
		if msgWithSigners, ok := msg.(interface{ GetSigners() []sdk.AccAddress }); ok {
			signers := msgWithSigners.GetSigners()
			for _, signer := range signers {
				addr := signer.String()
				
				entry, exists := rl.addressLimiters[addr]
				if !exists {
					rl.mu.RUnlock()
					rl.mu.Lock()
					entry, exists = rl.addressLimiters[addr]
					if !exists {
						entry = &addressEntry{
							limiter:  rate.NewLimiter(rl.perAddrRate, rl.burstSize),
							lastSeen: time.Now(),
						}
						rl.addressLimiters[addr] = entry
					}
					rl.mu.Unlock()
					rl.mu.RLock()
				}
				entry.lastSeen = time.Now()

				if !entry.limiter.Allow() {
					return fmt.Errorf("rate limit exceeded for address %s: %.0f tx/sec", addr, float64(rl.perAddrRate))
				}
			}
		}
	}
	
	return nil
}

// Cleanup removes address limiters that haven't been used within maxAge.
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	if rl == nil {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for addr, entry := range rl.addressLimiters {
		if entry.lastSeen.Before(cutoff) {
			delete(rl.addressLimiters, addr)
		}
	}
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	if rl == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}
	
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	return map[string]interface{}{
		"enabled":         true,
		"global_rate":     float64(rl.globalRate),
		"per_addr_rate":   float64(rl.perAddrRate),
		"burst_size":      rl.burstSize,
		"address_count":   len(rl.addressLimiters),
		"global_available": rl.globalLimiter.Tokens(),
	}
}

