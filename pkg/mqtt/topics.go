//nolint:goconst
package mqtt

import (
	"strings"

	iotago "github.com/iotaledger/iota.go/v4"
	"github.com/iotaledger/iota.go/v4/api"
	"github.com/iotaledger/iota.go/v4/hexutil"
)

func BlockIDFromBlockMetadataTopic(topic string) iotago.BlockID {
	if strings.HasPrefix(topic, "block-metadata/") && !(strings.HasSuffix(topic, api.EventAPITopicSuffixAccepted) || strings.HasSuffix(topic, api.EventAPITopicSuffixConfirmed)) {
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
	return api.EndpointWithNamedParameterValue(api.EventAPITopicBlocksBasicTaggedDataTag, api.ParameterTag, hexutil.EncodeHex(tag))
}

func GetTopicBlocksBasicTransactionTaggedDataTag(tag []byte) string {
	return api.EndpointWithNamedParameterValue(api.EventAPITopicBlocksBasicTransactionTaggedDataTag, api.ParameterTag, hexutil.EncodeHex(tag))
}

func GetTopicBlockMetadata(blockID iotago.BlockID) string {
	return api.EndpointWithNamedParameterValue(api.EventAPITopicBlockMetadata, api.ParameterBlockID, blockID.ToHex())
}

func GetTopicOutput(outputID iotago.OutputID) string {
	return api.EndpointWithNamedParameterValue(api.EventAPITopicOutputs, api.ParameterOutputID, outputID.ToHex())
}

func GetTopicTransactionsIncludedBlockMetadata(transactionID iotago.TransactionID) string {
	return api.EndpointWithNamedParameterValue(api.EventAPITopicTransactionsIncludedBlockMetadata, api.ParameterTransactionID, transactionID.ToHex())
}

func GetTopicTransactionMetadata(transactionID iotago.TransactionID) string {
	return api.EndpointWithNamedParameterValue(api.EventAPITopicTransactionMetadata, api.ParameterTransactionID, transactionID.ToHex())
}

func GetTopicAccountOutputs(accountID iotago.AccountID, hrp iotago.NetworkPrefix) string {
	return api.EndpointWithNamedParameterValue(api.EventAPITopicAccountOutputs, api.ParameterAccountAddress, accountID.ToAddress().Bech32(hrp))
}

func GetTopicAnchorOutputs(anchorID iotago.AnchorID, hrp iotago.NetworkPrefix) string {
	return api.EndpointWithNamedParameterValue(api.EventAPITopicAnchorOutputs, api.ParameterAnchorAddress, anchorID.ToAddress().Bech32(hrp))
}

func GetTopicFoundryOutputs(foundryID iotago.FoundryID) string {
	return api.EndpointWithNamedParameterValue(api.EventAPITopicFoundryOutputs, api.ParameterFoundryID, foundryID.ToHex())
}

func GetTopicNFTOutputs(nftID iotago.NFTID, hrp iotago.NetworkPrefix) string {
	return api.EndpointWithNamedParameterValue(api.EventAPITopicNFTOutputs, api.ParameterNFTAddress, nftID.ToAddress().Bech32(hrp))
}

func GetTopicDelegationOutputs(delegationID iotago.DelegationID) string {
	return api.EndpointWithNamedParameterValue(api.EventAPITopicDelegationOutputs, api.ParameterDelegationID, delegationID.ToHex())
}

func GetTopicOutputsByUnlockConditionAndAddress(condition api.EventAPIUnlockCondition, address iotago.Address, hrp iotago.NetworkPrefix) string {
	topic := api.EndpointWithNamedParameterValue(api.EventAPITopicOutputsByUnlockConditionAndAddress, api.ParameterCondition, string(condition))
	return api.EndpointWithNamedParameterValue(topic, api.ParameterAddress, address.Bech32(hrp))
}

func GetUnlockConditionTopicsForOutput(baseTopic string, output iotago.Output, bech32HRP iotago.NetworkPrefix) []string {
	topics := []string{}

	topicFunc := func(condition api.EventAPIUnlockCondition, addressString string) string {
		topic := api.EndpointWithNamedParameterValue(baseTopic, api.ParameterCondition, string(condition))
		return api.EndpointWithNamedParameterValue(topic, api.ParameterAddress, addressString)
	}

	unlockConditions := output.UnlockConditionSet()

	// this tracks the addresses used by any unlock condition
	// so that after checking all conditions we can see if anyone is subscribed to the wildcard
	addressesToPublishForAny := make(map[string]struct{})

	address := unlockConditions.Address()
	if address != nil {
		addr := address.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(api.EventAPIUnlockConditionAddress, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	storageReturn := unlockConditions.StorageDepositReturn()
	if storageReturn != nil {
		addr := storageReturn.ReturnAddress.Bech32(bech32HRP)
		topics = append(topics, topicFunc(api.EventAPIUnlockConditionStorageReturn, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	expiration := unlockConditions.Expiration()
	if expiration != nil {
		addr := expiration.ReturnAddress.Bech32(bech32HRP)
		topics = append(topics, topicFunc(api.EventAPIUnlockConditionExpiration, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	stateController := unlockConditions.StateControllerAddress()
	if stateController != nil {
		addr := stateController.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(api.EventAPIUnlockConditionStateController, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	governor := unlockConditions.GovernorAddress()
	if governor != nil {
		addr := governor.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(api.EventAPIUnlockConditionGovernor, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	immutableAccount := unlockConditions.ImmutableAccount()
	if immutableAccount != nil {
		addr := immutableAccount.Address.Bech32(bech32HRP)
		topics = append(topics, topicFunc(api.EventAPIUnlockConditionImmutableAccount, addr))
		addressesToPublishForAny[addr] = struct{}{}
	}

	for addr := range addressesToPublishForAny {
		topics = append(topics, topicFunc(api.EventAPIUnlockConditionAny, addr))
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
