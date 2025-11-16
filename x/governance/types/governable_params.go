package types

// GovernableParams defines which parameters can be changed through governance
// According to whitepaper:
// - Constitutional Level (Immutable): WRT/LZN tokenomics (total supply, halving schedule)
// - Legislative Level (Governable): Operational parameters (MOA coefficient, ANT limits, auction parameters)

// GovernableParameter represents a parameter that can be changed via governance
type GovernableParameter struct {
	Module    string // Module name (e.g., "lizenz", "anteil", "consensus")
	Parameter string // Parameter name
	Type      string // Parameter type (e.g., "string", "uint64", "duration")
}

// ConstitutionalParameters are parameters that CANNOT be changed via governance
// These are hardcoded in the protocol and can only be changed via hardfork
var ConstitutionalParameters = map[string]bool{
	// WRT tokenomics
	"wrt_total_supply":        true, // 21,000,000 WRT
	"wrt_halving_interval":    true, // 210,000 blocks
	"wrt_base_reward":         true, // 50 WRT per block (initial)
	
	// LZN tokenomics
	"lzn_total_supply":        true, // Fixed, one-time emission
	"lzn_denom":                true, // "ulzn" denomination
	
	// Core protocol rules
	"ident_zkp_requirement":   true, // ZKP verification requirement
	"ident_role_choice":        true, // Citizen/Validator role choice
	"lizenz_max_validator_share": true, // 33% limit (hardcoded)
}

// GovernableParameters lists all parameters that CAN be changed via governance
// According to whitepaper: "коэффициент MOA, лимит ANT, параметры аукциона"
var GovernableParameters = []GovernableParameter{
	// Lizenz module parameters
	{Module: "lizenz", Parameter: "activity_coefficient", Type: "string"}, // MOA coefficient
	{Module: "lizenz", Parameter: "min_lzn_amount", Type: "string"},
	{Module: "lizenz", Parameter: "max_lzn_amount", Type: "string"},
	{Module: "lizenz", Parameter: "inactivity_period", Type: "duration"},
	{Module: "lizenz", Parameter: "deactivation_period", Type: "duration"},
	
	// Anteil module parameters
	{Module: "anteil", Parameter: "min_ant_amount", Type: "string"}, // ANT limits
	{Module: "anteil", Parameter: "max_ant_amount", Type: "string"},
	{Module: "anteil", Parameter: "trading_fee_rate", Type: "string"},
	{Module: "anteil", Parameter: "max_open_orders", Type: "uint32"},
	
	// Consensus module parameters (auction parameters)
	{Module: "consensus", Parameter: "base_block_time", Type: "duration"},
	{Module: "consensus", Parameter: "high_activity_threshold", Type: "string"},
	{Module: "consensus", Parameter: "low_activity_threshold", Type: "string"},
	{Module: "consensus", Parameter: "min_validator_power", Type: "string"},
	{Module: "consensus", Parameter: "max_validator_power", Type: "string"},
	
	// Governance module parameters (meta-governance)
	{Module: "governance", Parameter: "voting_period", Type: "duration"},
	{Module: "governance", Parameter: "timelock_period", Type: "duration"},
	{Module: "governance", Parameter: "min_deposit", Type: "string"},
	{Module: "governance", Parameter: "quorum", Type: "string"},
	{Module: "governance", Parameter: "threshold", Type: "string"},
}

// IsGovernable checks if a parameter can be changed via governance
func IsGovernable(module, parameter string) bool {
	// Check if it's a constitutional parameter
	key := module + "_" + parameter
	if ConstitutionalParameters[key] {
		return false
	}
	
	// Check if it's in the list of governable parameters
	for _, gp := range GovernableParameters {
		if gp.Module == module && gp.Parameter == parameter {
			return true
		}
	}
	
	return false
}

// GetGovernableParameter returns the GovernableParameter for a given module and parameter
func GetGovernableParameter(module, parameter string) *GovernableParameter {
	for _, gp := range GovernableParameters {
		if gp.Module == module && gp.Parameter == parameter {
			return &gp
		}
	}
	return nil
}

