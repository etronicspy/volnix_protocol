package types

const (
	// EventTypeIdentityVerified defines the event type for identity verification
	EventTypeIdentityVerified = "ident.identity_verified"
	
	// EventTypeRoleMigrated defines the event type for role migration
	EventTypeRoleMigrated = "ident.role_migrated"
	
	// EventTypeActivityUpdated defines the event type for activity score update
	EventTypeActivityUpdated = "ident.activity_updated"
	
	// Attribute keys
	AttributeKeyAccount      = "account"
	AttributeKeyIdentityHash = "identity_hash"
	AttributeKeyRole         = "role"
	AttributeKeyOldRole      = "old_role"
	AttributeKeyNewRole      = "new_role"
	AttributeKeyActivityScore = "activity_score"
	AttributeKeyBlockHeight   = "block_height"
	AttributeKeyVerificationTime = "verification_time"
)

