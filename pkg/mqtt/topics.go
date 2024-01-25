//nolint:goconst
package mqtt

import (
	"strings"

	iotago "github.com/iotaledger/iota.go/v4"
	"github.com/iotaledger/iota.go/v4/hexutil"
)

// Topic names.
const (
	ParameterBlockID        = "{blockId}"
	ParameterTransactionID  = "{transactionId}"
	ParameterOutputID       = "{outputId}"
	ParameterTag            = "{tag}"
	ParameterAccountAddress = "{accountAddress}"
	ParameterAnchorAddress  = "{anchorAddress}"
	ParameterNFTAddress     = "{nftAddress}"
	ParameterFoundryID      = "{foundryId}"
	ParameterDelegationID   = "{delegationId}"
	ParameterAddress        = "{address}"
	ParameterCondition      = "{condition}"

	topicSuffixAccepted  = "accepted"
	topicSuffixConfirmed = "confirmed"

	// HINT: all existing topics always have a "/raw" suffix for the raw payload as well.
	TopicCommitmentsLatest    = "commitments/latest"    // iotago.Commitment
	TopicCommitmentsFinalized = "commitments/finalized" // iotago.Commitment

	TopicBlocks                              = "blocks"                                               // iotago.Block (track all incoming blocks)
	TopicBlocksValidation                    = "blocks/validation"                                    // iotago.Block (track all incoming validation blocks)
	TopicBlocksBasic                         = "blocks/basic"                                         // iotago.Block (track all incoming basic blocks)
	TopicBlocksBasicTaggedData               = "blocks/basic/tagged-data"                             // iotago.Block (track all incoming basic blocks with tagged data payload)
	TopicBlocksBasicTaggedDataTag            = "blocks/basic/tagged-data/" + ParameterTag             // iotago.Block (track all incoming basic blocks with specific tagged data payload)
	TopicBlocksBasicTransaction              = "blocks/basic/transaction"                             // iotago.Block (track all incoming basic blocks with transactions)
	TopicBlocksBasicTransactionTaggedData    = "blocks/basic/transaction/tagged-data"                 // iotago.Block (track all incoming basic blocks with transactions and tagged data)
	TopicBlocksBasicTransactionTaggedDataTag = "blocks/basic/transaction/tagged-data/" + ParameterTag // iotago.Block (track all incoming basic blocks with transactions and specific tagged data)

	// single block on subscribe and changes in it's metadata (accepted, confirmed).
	TopicTransactionsIncludedBlock = "transactions/" + ParameterTransactionID + "/included-block" // api.BlockWithMetadataResponse (track inclusion of a single transaction)
	TopicTransactionMetadata       = "transaction-metadata/" + ParameterTransactionID             // api.TransactionMetadataResponse (track a specific transaction)

	// single block on subscribe and changes in it's metadata (accepted, confirmed).
	TopicBlockMetadata = "block-metadata/" + ParameterBlockID // api.BlockMetadataResponse (track changes to a single block)

	// all blocks that arrive after subscribing.
	TopicBlockMetadataAccepted  = "block-metadata/" + topicSuffixAccepted  // api.BlockMetadataResponse (track acceptance of all blocks)
	TopicBlockMetadataConfirmed = "block-metadata/" + topicSuffixConfirmed // api.BlockMetadataResponse (track confirmation of all blocks)

	// single output on subscribe and changes in it's metadata (accepted, committed, spent).
	TopicOutputs = "outputs/" + ParameterOutputID // api.OutputWithMetadataResponse (track changes to a single output)

	// all outputs that arrive after subscribing (on transaction accepted and transaction committed).
	TopicAccountOutputs                     = "outputs/account/" + ParameterAccountAddress                    // api.OutputWithMetadataResponse (all changes of the chain output)
	TopicAnchorOutputs                      = "outputs/anchor/" + ParameterAnchorAddress                      // api.OutputWithMetadataResponse (all changes of the chain output)
	TopicFoundryOutputs                     = "outputs/foundry/" + ParameterFoundryID                         // api.OutputWithMetadataResponse (all changes of the chain output)
	TopicNFTOutputs                         = "outputs/nft/" + ParameterNFTAddress                            // api.OutputWithMetadataResponse (all changes of the chain output)
	TopicDelegationOutputs                  = "outputs/delegation/" + ParameterDelegationID                   // api.OutputWithMetadataResponse (all changes of the chain output)
	TopicOutputsByUnlockConditionAndAddress = "outputs/unlock/" + ParameterCondition + "/" + ParameterAddress // api.OutputWithMetadataResponse (all changes to outputs that match the unlock condition)
)

type UnlockCondition string

const (
	UnlockConditionAny              UnlockCondition = "+"
	UnlockConditionAddress          UnlockCondition = "address"
	UnlockConditionStorageReturn    UnlockCondition = "storage-return"
	UnlockConditionExpiration       UnlockCondition = "expiration"
	UnlockConditionStateController  UnlockCondition = "state-controller"
	UnlockConditionGovernor         UnlockCondition = "governor"
	UnlockConditionImmutableAccount UnlockCondition = "immutable-account"
)

func BlockIDFromBlockMetadataTopic(topic string) iotago.BlockID {
	if strings.HasPrefix(topic, "block-metadata/") && !(strings.HasSuffix(topic, topicSuffixAccepted) || strings.HasSuffix(topic, topicSuffixConfirmed)) {
		blockIDHex := strings.Replace(topic, "block-metadata/", "", 1)
		blockID, err := iotago.BlockIDFromHexString(blockIDHex)
		if err != nil {
			return iotago.EmptyBlockID
		}

		return blockID
	}

	return iotago.EmptyBlockID
}

func TransactionIDFromTransactionsIncludedBlockTopic(topic string) iotago.TransactionID {
	if strings.HasPrefix(topic, "transactions/") && strings.HasSuffix(topic, "/included-block") {
		transactionIDHex := strings.Replace(topic, "transactions/", "", 1)
		transactionIDHex = strings.Replace(transactionIDHex, "/included-block", "", 1)

		transactionID, err := iotago.TransactionIDFromHexString(transactionIDHex)
		if err != nil || len(transactionID) != iotago.TransactionIDLength {
			return iotago.EmptyTransactionID
		}

		return transactionID
	}

	return iotago.EmptyTransactionID
}

func TransactionIDFromTransactionMetadataTopic(topic string) iotago.TransactionID {
	if strings.HasPrefix(topic, "transaction-metadata/") && strings.Count(topic, "/") == 1 {
		transactionIDHex := strings.Replace(topic, "transaction-metadata/", "", 1)

		transactionID, err := iotago.TransactionIDFromHexString(transactionIDHex)
		if err != nil || len(transactionID) != iotago.TransactionIDLength {
			return iotago.EmptyTransactionID
		}

		return transactionID
	}

	return iotago.EmptyTransactionID
}

func OutputIDFromOutputsTopic(topic string) iotago.OutputID {
	if strings.HasPrefix(topic, "outputs/") && strings.Count(topic, "/") == 1 {
		outputIDHex := strings.Replace(topic, "outputs/", "", 1)
		outputID, err := iotago.OutputIDFromHexString(outputIDHex)
		if err != nil {
			return iotago.EmptyOutputID
		}

		return outputID
	}

	return iotago.EmptyOutputID
}

func GetTopicBlocksBasicTaggedDataTag(tag []byte) string {
	return strings.ReplaceAll(TopicBlocksBasicTaggedDataTag, ParameterTag, hexutil.EncodeHex(tag))
}

func GetTopicBlocksBasicTransactionTaggedDataTag(tag []byte) string {
	return strings.ReplaceAll(TopicBlocksBasicTransactionTaggedDataTag, ParameterTag, hexutil.EncodeHex(tag))
}

func GetTopicBlockMetadata(blockID iotago.BlockID) string {
	return strings.ReplaceAll(TopicBlockMetadata, ParameterBlockID, blockID.ToHex())
}

func GetTopicOutput(outputID iotago.OutputID) string {
	return strings.ReplaceAll(TopicOutputs, ParameterOutputID, outputID.ToHex())
}

func GetTopicTransactionsIncludedBlock(transactionID iotago.TransactionID) string {
	return strings.ReplaceAll(TopicTransactionsIncludedBlock, ParameterTransactionID, transactionID.ToHex())
}

func GetTopicTransactionMetadata(transactionID iotago.TransactionID) string {
	return strings.ReplaceAll(TopicTransactionMetadata, ParameterTransactionID, transactionID.ToHex())
}

func GetTopicAccountOutputs(accountID iotago.AccountID, hrp iotago.NetworkPrefix) string {
	return strings.ReplaceAll(TopicAccountOutputs, ParameterAccountAddress, accountID.ToAddress().Bech32(hrp))
}

func GetTopicAnchorOutputs(anchorID iotago.AnchorID, hrp iotago.NetworkPrefix) string {
	return strings.ReplaceAll(TopicAnchorOutputs, ParameterAnchorAddress, anchorID.ToAddress().Bech32(hrp))
}

func GetTopicFoundryOutputs(foundryID iotago.FoundryID) string {
	return strings.ReplaceAll(TopicFoundryOutputs, ParameterFoundryID, foundryID.ToHex())
}

func GetTopicNFTOutputs(nftID iotago.NFTID, hrp iotago.NetworkPrefix) string {
	return strings.ReplaceAll(TopicNFTOutputs, ParameterNFTAddress, nftID.ToAddress().Bech32(hrp))
}

func GetTopicDelegationOutputs(delegationID iotago.DelegationID) string {
	return strings.ReplaceAll(TopicDelegationOutputs, ParameterDelegationID, delegationID.ToHex())
}

func GetTopicOutputsByUnlockConditionAndAddress(condition UnlockCondition, address iotago.Address, hrp iotago.NetworkPrefix) string {
	topic := strings.ReplaceAll(TopicOutputsByUnlockConditionAndAddress, ParameterCondition, string(condition))
	return strings.ReplaceAll(topic, ParameterAddress, address.Bech32(hrp))
}

func GetUnlockConditionTopicsForOutput(baseTopic string, output iotago.Output, bech32HRP iotago.NetworkPrefix) []string {
	topics := []string{}

	topicFunc := func(condition UnlockCondition, addressString string) string {
		topic := strings.ReplaceAll(baseTopic, ParameterCondition, string(condition))
		return strings.ReplaceAll(topic, ParameterAddress, addressString)
	}

	unlockConditions := output.UnlockConditionSet()

	// this tracks the addresses used by any unlock condition
	// so that after checking all conditions we can see if anyone is subscribed to the wildcard
	addressesToPublishForAny := make(map[string]struct{})

	address := unlockConditions.Address()
	if address != nil {
		addr := address.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(UnlockConditionAddress, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	storageReturn := unlockConditions.StorageDepositReturn()
	if storageReturn != nil {
		addr := storageReturn.ReturnAddress.Bech32(bech32HRP)
		topics = append(topics, topicFunc(UnlockConditionStorageReturn, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	expiration := unlockConditions.Expiration()
	if expiration != nil {
		addr := expiration.ReturnAddress.Bech32(bech32HRP)
		topics = append(topics, topicFunc(UnlockConditionExpiration, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	stateController := unlockConditions.StateControllerAddress()
	if stateController != nil {
		addr := stateController.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(UnlockConditionStateController, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	governor := unlockConditions.GovernorAddress()
	if governor != nil {
		addr := governor.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(UnlockConditionGovernor, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	immutableAccount := unlockConditions.ImmutableAccount()
	if immutableAccount != nil {
		addr := immutableAccount.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(UnlockConditionImmutableAccount, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	for addr := range addressesToPublishForAny {
		topics = append(topics, topicFunc(UnlockConditionAny, addr))
	}

	return topics
}

func GetChainTopicsForOutput(outputID iotago.OutputID, output iotago.Output, bech32HRP iotago.NetworkPrefix) []string {
	topics := []string{}

	switch o := output.(type) {
	case *iotago.AccountOutput:
		accountID := o.AccountID
		if accountID.Empty() {
			// Use implicit AccountID
			accountID = iotago.AccountIDFromOutputID(outputID)
		}
		topics = append(topics, GetTopicAccountOutputs(accountID, bech32HRP))

	case *iotago.AnchorOutput:
		anchorID := o.AnchorID
		if anchorID.Empty() {
			// Use implicit AnchorID
			anchorID = iotago.AnchorIDFromOutputID(outputID)
		}
		topics = append(topics, GetTopicAnchorOutputs(anchorID, bech32HRP))

	case *iotago.FoundryOutput:
		foundryID, err := o.FoundryID()
		if err != nil {
			return topics
		}
		topics = append(topics, GetTopicFoundryOutputs(foundryID))

	case *iotago.NFTOutput:
		nftID := o.NFTID
		if nftID.Empty() {
			// Use implicit NFTID
			nftID = iotago.NFTIDFromOutputID(outputID)
		}
		topics = append(topics, GetTopicNFTOutputs(nftID, bech32HRP))

	case *iotago.DelegationOutput:
		delegationID := o.DelegationID
		if delegationID.Empty() {
			// Use implicit DelegationID
			delegationID = iotago.DelegationIDFromOutputID(outputID)
		}
		topics = append(topics, GetTopicDelegationOutputs(delegationID))

	default:
		// BasicOutput
	}

	return topics
}
