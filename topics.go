package main

// Topic names
const (
	topicMilestoneInfoLatest    = "milestone-info/latest"    // milestoneInfoPayload
	topicMilestoneInfoConfirmed = "milestone-info/confirmed" // milestoneInfoPayload
	topicMilestones             = "milestones"               // iotago.Milestone serialized => []bytes

	topicMessages                         = "messages"
	topicMessagesReferenced               = "messages/referenced"
	topicMessagesTransaction              = "messages/transaction"
	topicMessagesTransactionTaggedData    = "messages/transaction/tagged-data"
	topicMessagesTransactionTaggedDataTag = "messages/transaction/tagged-data/{tag}"
	topicMessagesTaggedData               = "messages/tagged-data"
	topicMessagesTaggedDataTag            = "messages/tagged-data/{tag}"
	topicMessagesMetadata                 = "messages/{messageId}/metadata"

	topicTransactionsIncludedMessage = "transactions/{transactionId}/included-message"

	topicOutputs                                 = "outputs/{outputId}"
	topicNFTOutputs                              = "outputs/nfts/{nftId}"
	topicAliasOutputs                            = "outputs/aliases/{aliasId}"
	topicFoundryOutputs                          = "outputs/foundries/{foundryId}"
	topicOutputsByUnlockConditionAndAddress      = "outputs/unlock/{condition}/{address}"
	topicSpentOutputsByUnlockConditionAndAddress = "outputs/unlock/{condition}/{address}/spent"

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
