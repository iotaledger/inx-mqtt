//nolint:goconst
package mqtt

import (
	"strings"

	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v4"
	"github.com/iotaledger/iota.go/v4/hexutil"
)

// Topic names.
const (
	parameterBlockID        = "{blockId}"
	parameterTransactionID  = "{transactionId}"
	parameterOutputID       = "{outputId}"
	parameterTag            = "{tag}"
	parameterAccountAddress = "{accountAddress}"
	parameterAnchorAddress  = "{anchorAddress}"
	parameterNFTAddress     = "{nftAddress}"
	parameterFoundryID      = "{foundryId}"
	parameterDelegationID   = "{delegationId}"
	parameterAddress        = "{address}"
	parameterCondition      = "{condition}"

	topicSuffixAccepted  = "accepted"
	topicSuffixConfirmed = "confirmed"

	// HINT: all existing topics always have a "/raw" suffix for the raw payload as well.
	topicCommitmentsLatest    = "commitments/latest"    // iotago.Commitment
	topicCommitmentsFinalized = "commitments/finalized" // iotago.Commitment

	topicBlocks                              = "blocks"                                               // iotago.Block (track all incoming blocks)
	topicBlocksValidation                    = "blocks/validation"                                    // iotago.Block (track all incoming validation blocks)
	topicBlocksBasic                         = "blocks/basic"                                         // iotago.Block (track all incoming basic blocks)
	topicBlocksBasicTransaction              = "blocks/basic/transaction"                             // iotago.Block (track all incoming basic blocks with transactions)
	topicBlocksBasicTransactionTaggedData    = "blocks/basic/transaction/tagged-data"                 // iotago.Block (track all incoming basic blocks with transactions and tagged data)
	topicBlocksBasicTransactionTaggedDataTag = "blocks/basic/transaction/tagged-data/" + parameterTag // iotago.Block (track all incoming basic blocks with transactions and specific tagged data)
	topicBlocksBasicTaggedData               = "blocks/basic/tagged-data"                             // iotago.Block (track all incoming basic blocks with tagged data payload)
	topicBlocksBasicTaggedDataTag            = "blocks/basic/tagged-data/" + parameterTag             // iotago.Block (track all incoming basic blocks with specific tagged data payload)

	topicTransactionsIncludedBlock = "transactions/" + parameterTransactionID + "/included-block" // apimodels.BlockWithMetadataResponse (track inclusion of a single transaction)

	// single block on subscribe and changes in it's metadata (accepted, confirmed).
	topicBlockMetadata = "block-metadata/" + parameterBlockID // apimodels.BlockMetadataResponse (track changes to a single block)

	// all blocks that arrive after subscribing.
	topicBlockMetadataAccepted  = "block-metadata/" + topicSuffixAccepted  // apimodels.BlockMetadataResponse (track acceptance of all blocks)
	topicBlockMetadataConfirmed = "block-metadata/" + topicSuffixConfirmed // apimodels.BlockMetadataResponse (track confirmation of all blocks)

	// single output on subscribe and changes in it's metadata (accepted, committed, spent).
	topicOutputs = "outputs/" + parameterOutputID // apimodels.OutputWithMetadataResponse (track changes to a single output)

	// all outputs that arrive after subscribing (on transaction accepted and transaction committed).
	topicAccountOutputs                     = "outputs/account/" + parameterAccountAddress                    // apimodels.OutputWithMetadataResponse (all changes of the chain output)
	topicAnchorOutputs                      = "outputs/anchor/" + parameterAnchorAddress                      // apimodels.OutputWithMetadataResponse (all changes of the chain output)
	topicFoundryOutputs                     = "outputs/foundry/" + parameterFoundryID                         // apimodels.OutputWithMetadataResponse (all changes of the chain output)
	topicNFTOutputs                         = "outputs/nft/" + parameterNFTAddress                            // apimodels.OutputWithMetadataResponse (all changes of the chain output)
	topicDelegationOutputs                  = "outputs/delegation/" + parameterDelegationID                   // apimodels.OutputWithMetadataResponse (all changes of the chain output)
	topicOutputsByUnlockConditionAndAddress = "outputs/unlock/" + parameterCondition + "/" + parameterAddress // apimodels.OutputWithMetadataResponse (all changes to outputs that match the unlock condition)

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

func blockIDFromBlockMetadataTopic(topic string) iotago.BlockID {
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

func transactionIDFromTransactionsIncludedBlockTopic(topic string) iotago.TransactionID {
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

func outputIDFromOutputsTopic(topic string) iotago.OutputID {
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

func getTopicBlocksBasicTransactionTaggedDataTag(tag []byte) string {
	return strings.ReplaceAll(topicBlocksBasicTransactionTaggedDataTag, parameterTag, hexutil.EncodeHex(tag))
}

func getTopicBlocksBasicTaggedDataTag(tag []byte) string {
	return strings.ReplaceAll(topicBlocksBasicTaggedDataTag, parameterTag, hexutil.EncodeHex(tag))
}

func getTopicBlockMetadata(blockID iotago.BlockID) string {
	return strings.ReplaceAll(topicBlockMetadata, parameterBlockID, blockID.ToHex())
}

func getTopicOutput(output *inx.LedgerOutput) string {
	return strings.ReplaceAll(topicOutputs, parameterOutputID, output.GetOutputId().Unwrap().ToHex())
}

func getTransactionsIncludedBlockTopic(transactionID iotago.TransactionID) string {
	return strings.ReplaceAll(topicTransactionsIncludedBlock, parameterTransactionID, transactionID.ToHex())
}

func getUnlockConditionTopicsForOutput(baseTopic string, output iotago.Output, bech32HRP iotago.NetworkPrefix) []string {
	topics := []string{}

	topicFunc := func(condition unlockCondition, addressString string) string {
		topic := strings.ReplaceAll(baseTopic, parameterCondition, string(condition))
		return strings.ReplaceAll(topic, parameterAddress, addressString)
	}

	unlockConditions := output.UnlockConditionSet()

	// this tracks the addresses used by any unlock condition
	// so that after checking all conditions we can see if anyone is subscribed to the wildcard
	addressesToPublishForAny := make(map[string]struct{})

	address := unlockConditions.Address()
	if address != nil {
		addr := address.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(unlockConditionAddress, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	storageReturn := unlockConditions.StorageDepositReturn()
	if storageReturn != nil {
		addr := storageReturn.ReturnAddress.Bech32(bech32HRP)
		topics = append(topics, topicFunc(unlockConditionStorageReturn, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	expiration := unlockConditions.Expiration()
	if expiration != nil {
		addr := expiration.ReturnAddress.Bech32(bech32HRP)
		topics = append(topics, topicFunc(unlockConditionExpiration, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	stateController := unlockConditions.StateControllerAddress()
	if stateController != nil {
		addr := stateController.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(unlockConditionStateController, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	governor := unlockConditions.GovernorAddress()
	if governor != nil {
		addr := governor.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(unlockConditionGovernor, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	immutableAccount := unlockConditions.ImmutableAccount()
	if immutableAccount != nil {
		addr := immutableAccount.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(unlockConditionImmutableAlias, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	for addr := range addressesToPublishForAny {
		topics = append(topics, topicFunc(unlockConditionAny, addr))
	}

	return topics
}

func getChainTopicsForOutput(outputID iotago.OutputID, output iotago.Output, bech32HRP iotago.NetworkPrefix) []string {
	topics := []string{}

	switch o := output.(type) {
	case *iotago.AccountOutput:
		accountID := o.AccountID
		if accountID.Empty() {
			// Use implicit AccountID
			accountID = iotago.AccountIDFromOutputID(outputID)
		}
		topics = append(topics, strings.ReplaceAll(topicAccountOutputs, parameterAccountAddress, accountID.ToAddress().Bech32(bech32HRP)))

	case *iotago.AnchorOutput:
		anchorID := o.AnchorID
		if anchorID.Empty() {
			// Use implicit AnchorID
			anchorID = iotago.AnchorIDFromOutputID(outputID)
		}
		topics = append(topics, strings.ReplaceAll(topicAnchorOutputs, parameterAnchorAddress, anchorID.ToAddress().Bech32(bech32HRP)))

	case *iotago.FoundryOutput:
		foundryID, err := o.FoundryID()
		if err != nil {
			return topics
		}
		topics = append(topics, strings.ReplaceAll(topicFoundryOutputs, parameterFoundryID, foundryID.String()))

	case *iotago.NFTOutput:
		nftID := o.NFTID
		if nftID.Empty() {
			// Use implicit NFTID
			nftID = iotago.NFTIDFromOutputID(outputID)
		}
		topics = append(topics, strings.ReplaceAll(topicNFTOutputs, parameterNFTAddress, nftID.ToAddress().Bech32(bech32HRP)))

	case *iotago.DelegationOutput:
		delegationID := o.DelegationID
		if delegationID.Empty() {
			// Use implicit DelegationID
			delegationID = iotago.DelegationIDFromOutputID(outputID)
		}
		topics = append(topics, strings.ReplaceAll(topicDelegationOutputs, parameterDelegationID, delegationID.String()))

	default:
		// BasicOutput
	}

	return topics
}
