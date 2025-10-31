package types

import (
	"fmt"
	"time"

	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewLizenz creates a new ActivatedLizenz instance (alias for backward compatibility)
func NewLizenz(validator string, amount string, identityHash string) *lizenzv1.ActivatedLizenz {
	return NewActivatedLizenz(validator, amount, identityHash)
}

// NewActivatedLizenz creates a new ActivatedLizenz instance
func NewActivatedLizenz(validator string, amount string, identityHash string) *lizenzv1.ActivatedLizenz {
	now := timestamppb.Now()
	return &lizenzv1.ActivatedLizenz{
		Validator:      validator,
		Amount:         amount,
		ActivationTime: now,
		LastActivity:   now,
		IdentityHash:   identityHash,
	}
}

// NewDeactivatingLizenz creates a new DeactivatingLizenz instance
func NewDeactivatingLizenz(validator string, amount string, reason string) *lizenzv1.DeactivatingLizenz {
	now := timestamppb.Now()
	deactivationEnd := timestamppb.New(now.AsTime().Add(24 * time.Hour)) // Default 24h deactivation period

	return &lizenzv1.DeactivatingLizenz{
		Validator:         validator,
		Amount:            amount,
		DeactivationStart: now,
		DeactivationEnd:   deactivationEnd,
		Reason:            reason,
	}
}

// NewMOAStatus creates a new MOAStatus instance
func NewMOAStatus(validator string, currentMOA string, requiredMOA string) *lizenzv1.MOAStatus {
	now := timestamppb.Now()
	nextCheck := timestamppb.New(now.AsTime().Add(24 * time.Hour)) // Default 24h check interval

	return &lizenzv1.MOAStatus{
		Validator:    validator,
		IsActive:     true,
		LastActivity: now,
		CurrentMoa:   currentMOA,
		RequiredMoa:  requiredMOA,
		NextCheck:    nextCheck,
		IsCompliant:  true,
	}
}

// IsActivatedLizenzValid checks if the activated LZN is valid
func IsActivatedLizenzValid(lizenz *lizenzv1.ActivatedLizenz) error {
	if lizenz.Validator == "" {
		return ErrEmptyValidator
	}
	if lizenz.Amount == "" {
		return ErrEmptyAmount
	}
	if lizenz.IdentityHash == "" {
		return ErrEmptyIdentityHash
	}
	return nil
}

// IsDeactivatingLizenzValid checks if the deactivating LZN is valid
func IsDeactivatingLizenzValid(lizenz *lizenzv1.DeactivatingLizenz) error {
	if lizenz.Validator == "" {
		return ErrEmptyValidator
	}
	if lizenz.Amount == "" {
		return ErrEmptyAmount
	}
	if lizenz.Reason == "" {
		return ErrEmptyReason
	}
	return nil
}

// IsMOAStatusValid checks if the MOA status is valid
func IsMOAStatusValid(status *lizenzv1.MOAStatus) error {
	if status.Validator == "" {
		return ErrEmptyValidator
	}
	if status.CurrentMoa == "" {
		return ErrEmptyCurrentMOA
	}
	if status.RequiredMoa == "" {
		return ErrEmptyRequiredMOA
	}
	return nil
}

// UpdateActivatedLizenzActivity updates the last activity timestamp
func UpdateActivatedLizenzActivity(lizenz *lizenzv1.ActivatedLizenz) {
	lizenz.LastActivity = timestamppb.Now()
}

// UpdateMOAStatusActivity updates the last activity timestamp and MOA values
func UpdateMOAStatusActivity(status *lizenzv1.MOAStatus, currentMOA string) {
	status.LastActivity = timestamppb.Now()
	status.CurrentMoa = currentMOA
	// Simple string comparison for compliance - in real implementation this would be decimal comparison
	status.IsCompliant = currentMOA >= status.RequiredMoa
}

// CalculateMOA calculates the MOA value based on activity data
func CalculateMOA(activityData string, params Params) (string, error) {
	// This is a simplified MOA calculation
	// In real implementation, this would use complex algorithms based on:
	// - Block production rate
	// - Transaction validation
	// - Network participation
	// - Time-based factors

	// For now, return a simplified MOA value based on activity data length
	if len(activityData) > 10 {
		return "85.5", nil // High activity
	} else if len(activityData) > 5 {
		return "65.0", nil // Medium activity
	}
	return "45.0", nil // Low activity
}
// CalculateLizenzPrice calculates the price of a license based on various factors
func CalculateLizenzPrice(licenseType string, duration int64) (string, error) {
	// Calculate license price based on multiple factors
	basePrice := 100.0 // Base price of 100 LZN
	
	// Factor 1: Market demand (simplified)
	demandMultiplier := 1.2 // 1.2x multiplier
	
	// Factor 2: License duration
	durationDays := float64(30) // Default 30 days
	if duration > 0 {
		durationDays = float64(duration)
	}
	durationMultiplier := durationDays / 30.0 // Scale by duration
	
	// Factor 3: License type premium
	typeMultiplier := 1.0
	switch licenseType {
	case "premium":
		typeMultiplier = 1.5 // 1.5x for premium
	case "enterprise":
		typeMultiplier = 2.0 // 2x for enterprise
	default:
		typeMultiplier = 1.0 // 1x for basic
	}
	
	// Calculate final price
	finalPrice := basePrice * demandMultiplier * durationMultiplier * typeMultiplier
	
	// Ensure minimum price
	minPrice := 10.0
	if finalPrice < minPrice {
		finalPrice = minPrice
	}
	
	return fmt.Sprintf("%.2f", finalPrice), nil
}