package main

// Topic names
const (
	parameterMessageID     = "{messageId}"
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

	topicMessages                         = "messages"                                         // iotago.Message serialized => []bytes
	topicMessagesReferenced               = "messages/referenced"                              // messageMetadataPayload
	topicMessagesTransaction              = "messages/transaction"                             // iotago.Message serialized => []bytes
	topicMessagesTransactionTaggedData    = "messages/transaction/tagged-data"                 // iotago.Message serialized => []bytes
	topicMessagesTransactionTaggedDataTag = "messages/transaction/tagged-data/" + parameterTag // iotago.Message serialized => []bytes
	topicMessagesTaggedData               = "messages/tagged-data"                             // iotago.Message serialized => []bytes
	topicMessagesTaggedDataTag            = "messages/tagged-data/" + parameterTag             // iotago.Message serialized => []bytes
	topicMessagesMetadata                 = "messages/" + parameterMessageID + "/metadata"     // messageMetadataPayload

	topicTransactionsIncludedMessage = "transactions/" + parameterTransactionID + "/included-message" // iotago.Message serialized => []bytes

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
