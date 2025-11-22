package app

import (
	"fmt"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/time/rate"
)

// RateLimiter provides rate limiting for transactions
type RateLimiter struct {
	// Global rate limiter for all transactions
	globalLimiter *rate.Limiter
	
	// Per-address rate limiters
	addressLimiters map[string]*rate.Limiter
	
	// Configuration
	globalRate  rate.Limit // Transactions per second globally
	perAddrRate rate.Limit // Transactions per second per address
	burstSize   int        // Burst size for rate limiting
	
	mu sync.RWMutex
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

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalRate:  1000.0, // 1000 tx/sec globally
		PerAddrRate: 10.0,   // 10 tx/sec per address
		BurstSize:   20,     // Allow bursts of 20 transactions
		Enabled:     true,
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if !config.Enabled {
		return nil // Return nil if rate limiting is disabled
	}
	
	rl := &RateLimiter{
		globalLimiter:   rate.NewLimiter(rate.Limit(config.GlobalRate), config.BurstSize),
		addressLimiters: make(map[string]*rate.Limiter),
		globalRate:       rate.Limit(config.GlobalRate),
		perAddrRate:      rate.Limit(config.PerAddrRate),
		burstSize:        config.BurstSize,
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
		return fmt.Errorf("global rate limit exceeded: %v tx/sec", rl.globalRate)
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
				
				// Get or create per-address limiter
				addrLimiter, exists := rl.addressLimiters[addr]
				if !exists {
					rl.mu.RUnlock()
					rl.mu.Lock()
					// Double-check after acquiring write lock
					addrLimiter, exists = rl.addressLimiters[addr]
					if !exists {
						addrLimiter = rate.NewLimiter(rl.perAddrRate, rl.burstSize)
						rl.addressLimiters[addr] = addrLimiter
					}
					rl.mu.Unlock()
					rl.mu.RLock()
				}
				
				if !addrLimiter.Allow() {
					return fmt.Errorf("rate limit exceeded for address %s: %v tx/sec", addr, rl.perAddrRate)
				}
			}
		}
	}
	
	return nil
}

// Cleanup removes old address limiters to prevent memory leaks
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	if rl == nil {
		return
	}
	
	// For now, we keep all limiters
	// In production, you might want to implement LRU cache or time-based cleanup
	// This is a simplified implementation
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

