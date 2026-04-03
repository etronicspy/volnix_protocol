package types

const (
	// §6.3 — module.event naming, snake_case
	EventTypeIdentityVerified = "ident.identity_verified"
	EventTypeRoleChanged      = "ident.role_changed"
	EventTypeRoleMigrated     = "ident.role_migrated"
	EventTypeActivityUpdated  = "ident.activity_updated"
	EventTypeProviderChanged  = "ident.provider_changed"
	EventTypeAntBurned        = "ident.ant_burned" // unified burn event

	// §6.3 mandatory attributes
	AttributeKeyAccount          = "account"
	AttributeKeyBlockHeight      = "block_height"
	AttributeKeyTxHash           = "tx_hash"
	AttributeKeyIdentityHash     = "identity_hash"
	AttributeKeyOldRole          = "old_role"
	AttributeKeyNewRole          = "new_role"
	AttributeKeyRoleChangeReason = "role_change_reason"
	AttributeKeyBurnAmount       = "burn_amount"
	AttributeKeyBurnReason       = "burn_reason"
	AttributeKeyProviderID       = "provider_id"
	AttributeKeyAccreditStatus   = "accreditation_status"

	// §6.3 / Appendix A — role_change_reason values
	RoleChangeReasonZKPVerify         = "zkp_verify"
	RoleChangeReasonRoleChange        = "role_change"
	RoleChangeReasonRoleMigration     = "role_migration"
	RoleChangeReasonMOASupplier       = "moa_supplier"
	RoleChangeReasonMOAValidator      = "moa_validator"
	RoleChangeReasonGovernance        = "governance"
	RoleChangeReasonDeactivationOther = "deactivation_other"

	// §6.3 / Appendix A — burn_reason values (shared across modules)
	BurnReasonPerHeight          = "per_height"
	BurnReasonMOASupplier        = "moa_supplier"
	BurnReasonMOAValidator       = "moa_validator"
	BurnReasonEpochSupplierReset = "epoch_supplier_reset"
	BurnReasonProtocolOther      = "protocol_other"

	// Role string values for events (§6.3)
	RoleGuest     = "guest"
	RoleSupplier  = "supplier"
	RoleValidator = "validator"
)
