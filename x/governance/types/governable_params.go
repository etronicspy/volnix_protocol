package types

// GovernableParameter represents a parameter that can be changed via governance §7.2
type GovernableParameter struct {
	Module    string
	Parameter string
	Type      string
}

// ConstitutionalParameters — CANNOT be changed via governance, only hardfork §7.1
var ConstitutionalParameters = map[string]bool{
	"wrt_total_supply":     true,
	"wrt_halving_interval": true,
	"wrt_base_reward":      true,

	"lzn_total_supply": true,
	"lzn_denom":        true,

	"ident_zkp_requirement":          true,
	"ident_role_choice":              true,
	"lizenz_max_validator_share":     true, // 33%

	// λ bounds are constitutional §7.1; λ value itself is DAO §7.2 п.10
	"consensus_burn_cap_lambda_min": true, // 1/3
	"consensus_burn_cap_lambda_max": true, // 2/3

	// PoVB structure: s_i + b_i ≤ L_i, 1 LZN = 1 ANT
	"consensus_povb_structure": true,
	// ANT only through internal market
	"anteil_ant_direct_transfer_ban": true,
	// No hard max supply for ANT
	"anteil_ant_no_max_supply": true,
}

// GovernableParameters — exhaustive §7.2 list (10 params + governance meta)
var GovernableParameters = []GovernableParameter{
	// §7.2 п.1 — LZN freeze period after activation
	{Module: "lizenz", Parameter: "activation_freeze_period", Type: "duration"},
	{Module: "lizenz", Parameter: "min_lzn_amount", Type: "string"},
	{Module: "lizenz", Parameter: "max_lzn_amount", Type: "string"},

	// §7.2 п.2 — MOA Supplier window T_g
	{Module: "ident", Parameter: "moa_supplier_window", Type: "duration"},

	// §7.2 п.3 — MOA Validator window T_v
	{Module: "ident", Parameter: "moa_validator_window", Type: "duration"},

	// §7.2 п.4 — Supplier per-epoch ANT accumulation limit
	{Module: "anteil", Parameter: "supplier_epoch_ant_limit", Type: "string"},

	// §7.2 п.5 — Epoch emission coefficient bounds
	{Module: "anteil", Parameter: "epoch_coefficient_min", Type: "string"},
	{Module: "anteil", Parameter: "epoch_coefficient_max", Type: "string"},

	// §7.2 п.6 — Fee distribution policy when B=0
	{Module: "consensus", Parameter: "fee_policy_b_zero", Type: "string"},

	// §7.2 п.7 — Max gas per block
	{Module: "consensus", Parameter: "max_block_gas", Type: "uint64"},

	// §7.2 п.8 — Max block size in bytes
	{Module: "consensus", Parameter: "max_block_bytes", Type: "uint64"},

	// §7.2 п.9 — Max active Suppliers
	{Module: "ident", Parameter: "max_active_suppliers", Type: "uint64"},

	// §7.2 п.10 — λ (lambda) for global burn cap C = λ · L_tot
	{Module: "consensus", Parameter: "burn_cap_lambda", Type: "string"},

	// Anteil operational
	{Module: "anteil", Parameter: "trading_fee_rate", Type: "string"},
	{Module: "anteil", Parameter: "max_open_orders", Type: "uint32"},

	// Governance meta-governance
	{Module: "governance", Parameter: "voting_period", Type: "duration"},
	{Module: "governance", Parameter: "timelock_period", Type: "duration"},
	{Module: "governance", Parameter: "min_deposit", Type: "string"},
	{Module: "governance", Parameter: "quorum", Type: "string"},
	{Module: "governance", Parameter: "threshold", Type: "string"},
}

// IsGovernable checks if a parameter can be changed via governance
func IsGovernable(module, parameter string) bool {
	key := module + "_" + parameter
	if ConstitutionalParameters[key] {
		return false
	}
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
