package mqtt

// Topic names
const (
	parameterBlockID       = "{blockId}"
	parameterTransactionID = "{transactionId}"
	parameterOutputID      = "{outputId}"
	parameterTag           = "{tag}"
	parameterNFTID         = "{nftId}"
	parameterAliasID       = "{aliasId}"
	parameterFoundryID     = "{foundryId}"
	parameterCondition     = "{condition}"
	parameterAddress       = "{address}"

	topicMilestoneInfoLatest    = "milestone-info/latest"    // milestoneInfoPayload
	topicMilestoneInfoConfirmed = "milestone-info/confirmed" // milestoneInfoPayload
	topicMilestones             = "milestones"               // iotago.Milestone serialized => []bytes

	topicBlocks                         = "blocks"                                         // iotago.Block serialized => []bytes
	topicBlocksTransaction              = "blocks/transaction"                             // iotago.Block serialized => []bytes
	topicBlocksTransactionTaggedData    = "blocks/transaction/tagged-data"                 // iotago.Block serialized => []bytes
	topicBlocksTransactionTaggedDataTag = "blocks/transaction/tagged-data/" + parameterTag // iotago.Block serialized => []bytes
	topicBlocksTaggedData               = "blocks/tagged-data"                             // iotago.Block serialized => []bytes
	topicBlocksTaggedDataTag            = "blocks/tagged-data/" + parameterTag             // iotago.Block serialized => []bytes

	topicTransactionsIncludedBlock = "transactions/" + parameterTransactionID + "/included-block" // iotago.Block serialized => []bytes

	topicBlockMetadata           = "block-metadata/" + parameterBlockID // blockMetadataPayload	// renotify if "reattach" or "promote" changes? => add new INX event?
	topicBlockMetadataReferenced = "block-metadata/referenced"          // blockMetadataPayload

	topicOutputs                                 = "outputs/" + parameterOutputID                                             // outputPayload
	topicNFTOutputs                              = "outputs/nfts/" + parameterNFTID                                           // outputPayload
	topicAliasOutputs                            = "outputs/aliases/" + parameterAliasID                                      // outputPayload
	topicFoundryOutputs                          = "outputs/foundries/" + parameterFoundryID                                  // outputPayload
	topicOutputsByUnlockConditionAndAddress      = "outputs/unlock/" + parameterCondition + "/" + parameterAddress            // outputPayload
	topicSpentOutputsByUnlockConditionAndAddress = "outputs/unlock/" + parameterCondition + "/" + parameterAddress + "/spent" // outputPayload

	topicReceipts = "receipts"
)

type unlockCondition string

const (
	unlockConditionAny             unlockCondition = "+"
	unlockConditionAddress         unlockCondition = "address"
	unlockConditionStorageReturn   unlockCondition = "storage-return"
	unlockConditionExpiration      unlockCondition = "expiration"
	unlockConditionStateController unlockCondition = "state-controller"
	unlockConditionGovernor        unlockCondition = "governor"
	unlockConditionImmutableAlias  unlockCondition = "immutable-alias"
)
