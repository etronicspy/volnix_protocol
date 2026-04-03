package types

const (
	// §6.3 — consensus module events
	EventTypeBurnExecuted         = "consensus.burn_executed"
	EventTypeRewardDistributed    = "consensus.reward_distributed"
	EventTypeHalving              = "consensus.halving"
	EventTypeConsensusStateUpdate = "consensus.consensus_state_updated"
	EventTypeValidatorPowerUpdate = "consensus.validator_power_updated"
	EventTypeFeeDistributed       = "consensus.fee_distributed"
	EventTypePerHeightBurn        = "consensus.per_height_burn"

	// §6.3 mandatory attributes
	AttributeKeyValidator           = "validator"
	AttributeKeyBlockHeight         = "block_height"
	AttributeKeyTxHash              = "tx_hash"
	AttributeKeyAccount             = "account"
	AttributeKeyConsensusAddress    = "consensus_address"
	AttributeKeyPower               = "power"
	AttributeKeyBurnAmount          = "burn_amount"
	AttributeKeyBurnReason          = "burn_reason"
	AttributeKeyRewardAmount        = "reward_amount"
	AttributeKeyRewardShare         = "reward_share"
	AttributeKeyMOACompliance       = "moa_compliance"
	AttributeKeyHeight              = "height"
	AttributeKeyNextHalving         = "next_halving"

	// PoVB §5.4 specific attributes
	AttributeKeySi    = "s_i"    // priority stake
	AttributeKeyBi    = "b_i"    // burn for fee share
	AttributeKeyIncI  = "inc_i"  // included after global cap
	AttributeKeyLTot  = "l_tot"  // total activated LZN
	AttributeKeyLambda = "lambda" // global burn cap coefficient
	AttributeKeyTotalFees = "total_fees" // F
	AttributeKeyTotalB    = "total_b"    // B = Σ inc_i

	// §6.3 / Appendix A — consensus_change_reason values
	ConsensusChangeReasonLZNActivated   = "lzn_activated"
	ConsensusChangeReasonLZNDeactivated = "lzn_deactivated"
	ConsensusChangeReasonMOAValidator   = "moa_validator"
	ConsensusChangeReasonGovernance     = "governance"
	ConsensusChangeReasonSlashingOther  = "slashing_other"
	ConsensusChangeReasonGenesisInit    = "genesis_init"

	AttributeKeyConsensusChangeReason = "consensus_change_reason"

	// Shared burn_reason values
	BurnReasonPerHeight          = "per_height"
	BurnReasonMOASupplier        = "moa_supplier"
	BurnReasonMOAValidator       = "moa_validator"
	BurnReasonEpochSupplierReset = "epoch_supplier_reset"
	BurnReasonProtocolOther      = "protocol_other"

	// §7.2 п.6 — fee policy when B=0
	FeePolicyBZeroCommunityPool = "community_pool"
	FeePolicyBZeroCarryForward  = "carry_forward"
)
