package types

import (
	"crypto/rand"
	"fmt"
	"time"

	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"
	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
)

// ModuleIntegration represents the integration status between modules
type ModuleIntegration struct {
	ModuleName    string    `json:"module_name"`
	Status        string    `json:"status"`
	LastSync      time.Time `json:"last_sync"`
	Dependencies  []string  `json:"dependencies"`
	HealthScore   int64     `json:"health_score"`
	ErrorCount    int64     `json:"error_count"`
	LastError     string    `json:"last_error"`
}

// CrossModuleEvent represents events that affect multiple modules
type CrossModuleEvent struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	SourceModule  string    `json:"source_module"`
	TargetModule  string    `json:"target_module"`
	EventData     string    `json:"event_data"`
	Timestamp     time.Time `json:"timestamp"`
	Validator     string    `json:"validator"`
	RequiresAction bool     `json:"requires_action"`
}

// ValidatorIntegrationStatus represents the complete integration status of a validator
type ValidatorIntegrationStatus struct {
	Validator     string                    `json:"validator"`
	IdentStatus  *identv1.VerifiedAccount  `json:"ident_status"`
	LizenzStatus *lizenzv1.ActivatedLizenz `json:"lizenz_status"`
	AnteilAccess *anteilv1.UserPosition    `json:"anteil_access"`
	ConsensusRole *consensusv1.Validator   `json:"consensus_role"`
	OverallScore string                    `json:"overall_score"`
	LastUpdate   time.Time                 `json:"last_update"`
}

// IntegrationManager handles cross-module operations
type IntegrationManager struct {
	Modules map[string]*ModuleIntegration
	Events  []*CrossModuleEvent
}

// NewIntegrationManager creates a new integration manager
func NewIntegrationManager() *IntegrationManager {
	return &IntegrationManager{
		Modules: make(map[string]*ModuleIntegration),
		Events:  []*CrossModuleEvent{},
	}
}

// RegisterModule registers a module for integration
func (im *IntegrationManager) RegisterModule(name string, dependencies []string) {
	im.Modules[name] = &ModuleIntegration{
		ModuleName:   name,
		Status:       "active",
		LastSync:     time.Now(),
		Dependencies: dependencies,
		HealthScore:  100,
		ErrorCount:   0,
	}
}

// UpdateModuleHealth updates the health status of a module
func (im *IntegrationManager) UpdateModuleHealth(name string, healthScore int64, errorMsg string) {
	if module, exists := im.Modules[name]; exists {
		module.HealthScore = healthScore
		module.LastSync = time.Now()
		if errorMsg != "" {
			module.ErrorCount++
			module.LastError = errorMsg
		}
	}
}

// AddCrossModuleEvent adds a new cross-module event
func (im *IntegrationManager) AddCrossModuleEvent(eventType, sourceModule, targetModule, eventData, validator string) {
	event := &CrossModuleEvent{
		EventID:       generateEventID(),
		EventType:     eventType,
		SourceModule:  sourceModule,
		TargetModule:  targetModule,
		EventData:     eventData,
		Timestamp:     time.Now(),
		Validator:     validator,
		RequiresAction: false,
	}
	im.Events = append(im.Events, event)
}

// GetValidatorIntegrationStatus gets the complete integration status for a validator
func GetValidatorIntegrationStatus(
	validator string,
	identAccount *identv1.VerifiedAccount,
	lizenzLicense *lizenzv1.ActivatedLizenz,
	anteilPosition *anteilv1.UserPosition,
	consensusValidator *consensusv1.Validator,
) *ValidatorIntegrationStatus {
	
	// Calculate overall score based on all module statuses
	overallScore := calculateOverallScore(identAccount, lizenzLicense, anteilPosition, consensusValidator)
	
	return &ValidatorIntegrationStatus{
		Validator:     validator,
		IdentStatus:  identAccount,
		LizenzStatus: lizenzLicense,
		AnteilAccess: anteilPosition,
		ConsensusRole: consensusValidator,
		OverallScore: overallScore,
		LastUpdate:   time.Now(),
	}
}

// calculateOverallScore calculates the overall integration score for a validator
func calculateOverallScore(
	identAccount *identv1.VerifiedAccount,
	lizenzLicense *lizenzv1.ActivatedLizenz,
	anteilPosition *anteilv1.UserPosition,
	consensusValidator *consensusv1.Validator,
) string {
	
	// This is a placeholder implementation
	// In a real implementation, you would calculate based on:
	// - Identity verification status
	// - LZN license activation
	// - ANT market participation
	// - Consensus contribution
	
	score := 0.0
	
	if identAccount != nil && identAccount.IsActive {
		score += 25.0
	}
	
	if lizenzLicense != nil {
		score += 25.0
	}
	
	if anteilPosition != nil {
		score += 25.0
	}
	
	if consensusValidator != nil {
		score += 25.0
	}
	
	return fmt.Sprintf("%.2f", score)
}

// generateEventID generates a unique event ID
func generateEventID() string {
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	return time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%x", randomBytes)
}
