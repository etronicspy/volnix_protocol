package types

const (
	// EventTypeBlockCreatorSelected defines the event type for block creator selection
	EventTypeBlockCreatorSelected = "consensus.block_creator_selected"
	
	// EventTypeValidatorPowerUpdated defines the event type for validator power updates
	EventTypeValidatorPowerUpdated = "consensus.validator_power_updated"
	
	// EventTypeBlockTimeAdjusted defines the event type for block time adjustments
	EventTypeBlockTimeAdjusted = "consensus.block_time_adjusted"
	
	// EventTypeHalving defines the event type for halving events
	EventTypeHalving = "consensus.halving"
	
	// EventTypeConsensusStateUpdated defines the event type for consensus state updates
	EventTypeConsensusStateUpdated = "consensus.consensus_state_updated"
	
	// EventTypeBurnExecuted defines the event type for ANT token burning
	EventTypeBurnExecuted = "consensus.burn_executed"
	
	// EventTypeRewardDistributed defines the event type for WRT reward distribution
	EventTypeRewardDistributed = "consensus.reward_distributed"
	
	// EventTypeAuctionStarted defines the event type for blind auction start
	EventTypeAuctionStarted = "consensus.auction_started"
	
	// EventTypeAuctionCompleted defines the event type for blind auction completion
	EventTypeAuctionCompleted = "consensus.auction_completed"
	
	// EventTypeBidCommitted defines the event type for bid commit in blind auction
	EventTypeBidCommitted = "consensus.bid_committed"
	
	// EventTypeBidRevealed defines the event type for bid reveal in blind auction
	EventTypeBidRevealed = "consensus.bid_revealed"
	
	// Attribute keys
	AttributeKeyBlockCreator = "block_creator"
	AttributeKeyBlockHeight  = "block_height"
	AttributeKeyValidator    = "validator"
	AttributeKeyPower        = "power"
	AttributeKeyBlockTime    = "block_time"
	AttributeKeyHeight       = "height"
	AttributeKeyNextHalving  = "next_halving"
	AttributeKeyBurnAmount   = "burn_amount"
	AttributeKeyNewBalance   = "new_balance"
	AttributeKeyAuctionWinner = "auction_winner"
	AttributeKeyRewardAmount = "reward_amount"
	AttributeKeyRewardShare   = "reward_share"
	AttributeKeyMOACompliance = "moa_compliance"
	AttributeKeyPenaltyMultiplier = "penalty_multiplier"
	AttributeKeyAuctionHeight = "auction_height"
	AttributeKeyCommitHash   = "commit_hash"
	AttributeKeyBidAmount    = "bid_amount"
	AttributeKeyNonce        = "nonce"
)
