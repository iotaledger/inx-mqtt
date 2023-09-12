package mqtt

import (
	"context"
	"encoding/json"
	"strings"

	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v4"
	"github.com/iotaledger/iota.go/v4/hexutil"
)

func (s *Server) sendMessageOnTopic(topic string, payload []byte) {
	if err := s.MQTTBroker.Send(topic, payload); err != nil {
		s.LogWarnf("Failed to send message on topic %s: %s", topic, err)
	}
}

func (s *Server) PublishRawOnTopicIfSubscribed(topic string, payload []byte) {
	if s.MQTTBroker.HasSubscribers(topic) {
		s.sendMessageOnTopic(topic, payload)
	}
}

func (s *Server) PublishPayloadFuncOnTopicIfSubscribed(topic string, payloadFunc func() interface{}) {
	if s.MQTTBroker.HasSubscribers(topic) {
		s.PublishOnTopic(topic, payloadFunc())
	}
}

func (s *Server) PublishOnTopicIfSubscribed(topic string, payload interface{}) {
	if s.MQTTBroker.HasSubscribers(topic) {
		s.PublishOnTopic(topic, payload)
	}
}

func (s *Server) PublishOnTopic(topic string, payload interface{}) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return
	}

	s.sendMessageOnTopic(topic, jsonPayload)
}

func (s *Server) PublishRawCommitmentOnTopic(topic string, commitment *iotago.Commitment) {
	apiForVersion, err := s.NodeBridge.APIProvider().APIForVersion(commitment.Version)
	if err != nil {
		return
	}

	rawCommitment, err := apiForVersion.Encode(commitment)
	if err != nil {
		return
	}

	s.PublishRawOnTopicIfSubscribed(topic, rawCommitment)
}

func (s *Server) PublishCommitmentInfoOnTopic(topic string, id iotago.CommitmentID) {
	s.PublishOnTopicIfSubscribed(topic, &commitmentInfoPayload{
		CommitmentID:    id.ToHex(),
		CommitmentIndex: uint64(id.Index()),
	})
}

func (s *Server) PublishBlock(blk *inx.RawBlock) {
	version, _, err := iotago.VersionFromBytes(blk.GetData())
	if err != nil {
		return
	}

	apiForVersion, err := s.NodeBridge.APIProvider().APIForVersion(version)
	if err != nil {
		return
	}

	block, err := blk.UnwrapBlock(apiForVersion)
	if err != nil {
		return
	}

	s.PublishRawOnTopicIfSubscribed(topicBlocks, blk.GetData())

	basicBlk, isBasicBlk := block.Block.(*iotago.BasicBlock)
	if !isBasicBlk {
		return
	}

	switch payload := basicBlk.Payload.(type) {
	case *iotago.Transaction:
		s.PublishRawOnTopicIfSubscribed(topicBlocksTransaction, blk.GetData())

		//nolint:gocritic // the type switch is nicer here
		switch p := payload.Essence.Payload.(type) {
		case *iotago.TaggedData:
			s.PublishRawOnTopicIfSubscribed(topicBlocksTransactionTaggedData, blk.GetData())
			if len(p.Tag) > 0 {
				txTaggedDataTagTopic := strings.ReplaceAll(topicBlocksTransactionTaggedDataTag, parameterTag, hexutil.EncodeHex(p.Tag))
				s.PublishRawOnTopicIfSubscribed(txTaggedDataTagTopic, blk.GetData())
			}
		}

	case *iotago.TaggedData:
		s.PublishRawOnTopicIfSubscribed(topicBlocksTaggedData, blk.GetData())
		if len(payload.Tag) > 0 {
			taggedDataTagTopic := strings.ReplaceAll(topicBlocksTaggedDataTag, parameterTag, hexutil.EncodeHex(payload.Tag))
			s.PublishRawOnTopicIfSubscribed(taggedDataTagTopic, blk.GetData())
		}
	}
}

func (s *Server) hasSubscriberForTransactionIncludedBlock(transactionID iotago.TransactionID) bool {
	transactionTopic := strings.ReplaceAll(topicTransactionsIncludedBlock, parameterTransactionID, transactionID.ToHex())

	return s.MQTTBroker.HasSubscribers(transactionTopic)
}

func (s *Server) PublishTransactionIncludedBlock(transactionID iotago.TransactionID, block *inx.RawBlock) {
	transactionTopic := strings.ReplaceAll(topicTransactionsIncludedBlock, parameterTransactionID, transactionID.ToHex())
	s.PublishRawOnTopicIfSubscribed(transactionTopic, block.GetData())
}

func (s *Server) PublishBlockMetadata(metadata *inx.BlockMetadata) {
	blockID := metadata.GetBlockId().Unwrap().ToHex()
	singleBlockTopic := strings.ReplaceAll(topicBlockMetadata, parameterBlockID, blockID)
	hasSingleBlockTopicSubscriber := s.MQTTBroker.HasSubscribers(singleBlockTopic)
	hasAcceptedBlocksTopicSubscriber := s.MQTTBroker.HasSubscribers(topicBlockMetadataAccepted)
	hasConfirmedBlocksTopicSubscriber := s.MQTTBroker.HasSubscribers(topicBlockMetadataConfirmed)

	if !hasSingleBlockTopicSubscriber && !hasConfirmedBlocksTopicSubscriber && !hasAcceptedBlocksTopicSubscriber {
		return
	}

	response := &blockMetadataPayload{
		BlockID:            blockID,
		BlockState:         metadata.GetBlockState(),
		BlockFailureReason: metadata.GetBlockFailureReason(),
		TxState:            metadata.GetTxState(),
		TxFailureReason:    metadata.GetTxFailureReason(),
	}

	// Serialize here instead of using publishOnTopic to avoid double JSON marshaling
	jsonPayload, err := json.Marshal(response)
	if err != nil {
		return
	}

	if hasSingleBlockTopicSubscriber {
		s.sendMessageOnTopic(singleBlockTopic, jsonPayload)
	}
	if hasConfirmedBlocksTopicSubscriber {
		s.sendMessageOnTopic(topicBlockMetadataConfirmed, jsonPayload)
	}
	if hasAcceptedBlocksTopicSubscriber {
		s.sendMessageOnTopic(topicBlockMetadataAccepted, jsonPayload)
	}
}

func payloadForOutput(api iotago.API, output *inx.LedgerOutput, iotaOutput iotago.Output) *outputPayload {
	rawOutputJSON, err := api.JSONEncode(iotaOutput)
	if err != nil {
		return nil
	}

	rawRawOutputJSON := json.RawMessage(rawOutputJSON)

	return &outputPayload{
		RawOutput: &rawRawOutputJSON,
	}
}

func payloadForSpent(api iotago.API, spent *inx.LedgerSpent, iotaOutput iotago.Output) *outputPayload {
	return payloadForOutput(api, spent.GetOutput(), iotaOutput)
}

func (s *Server) PublishOnUnlockConditionTopics(baseTopic string, output iotago.Output, payloadFunc func() interface{}) {

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
		addr := address.Address.Bech32(s.NodeBridge.APIProvider().CurrentAPI().ProtocolParameters().Bech32HRP())
		s.PublishPayloadFuncOnTopicIfSubscribed(topicFunc(unlockConditionAddress, addr), payloadFunc)
		addressesToPublishForAny[addr] = struct{}{}
	}

	storageReturn := unlockConditions.StorageDepositReturn()
	if storageReturn != nil {
		addr := storageReturn.ReturnAddress.Bech32(s.NodeBridge.APIProvider().CurrentAPI().ProtocolParameters().Bech32HRP())
		s.PublishPayloadFuncOnTopicIfSubscribed(topicFunc(unlockConditionStorageReturn, addr), payloadFunc)
		addressesToPublishForAny[addr] = struct{}{}
	}

	expiration := unlockConditions.Expiration()
	if expiration != nil {
		addr := expiration.ReturnAddress.Bech32(s.NodeBridge.APIProvider().CurrentAPI().ProtocolParameters().Bech32HRP())
		s.PublishPayloadFuncOnTopicIfSubscribed(topicFunc(unlockConditionExpiration, addr), payloadFunc)
		addressesToPublishForAny[addr] = struct{}{}
	}

	stateController := unlockConditions.StateControllerAddress()
	if stateController != nil {
		addr := stateController.Address.Bech32(s.NodeBridge.APIProvider().CurrentAPI().ProtocolParameters().Bech32HRP())
		s.PublishPayloadFuncOnTopicIfSubscribed(topicFunc(unlockConditionStateController, addr), payloadFunc)
		addressesToPublishForAny[addr] = struct{}{}
	}

	governor := unlockConditions.GovernorAddress()
	if governor != nil {
		addr := governor.Address.Bech32(s.NodeBridge.APIProvider().CurrentAPI().ProtocolParameters().Bech32HRP())
		s.PublishPayloadFuncOnTopicIfSubscribed(topicFunc(unlockConditionGovernor, addr), payloadFunc)
		addressesToPublishForAny[addr] = struct{}{}
	}

	immutableAccount := unlockConditions.ImmutableAccount()
	if immutableAccount != nil {
		addr := immutableAccount.Address.Bech32(s.NodeBridge.APIProvider().CurrentAPI().ProtocolParameters().Bech32HRP())
		s.PublishPayloadFuncOnTopicIfSubscribed(topicFunc(unlockConditionImmutableAlias, addr), payloadFunc)
		addressesToPublishForAny[addr] = struct{}{}
	}

	for addr := range addressesToPublishForAny {
		s.PublishPayloadFuncOnTopicIfSubscribed(topicFunc(unlockConditionAny, addr), payloadFunc)
	}
}

func (s *Server) PublishOnOutputChainTopics(outputID iotago.OutputID, output iotago.Output, payloadFunc func() interface{}) {

	switch o := output.(type) {
	case *iotago.NFTOutput:
		nftID := o.NFTID
		if nftID.Empty() {
			// Use implicit NFTID
			nftAddr := iotago.NFTAddressFromOutputID(outputID)
			nftID = nftAddr.NFTID()
		}
		topic := strings.ReplaceAll(topicNFTOutputs, parameterNFTID, nftID.String())
		s.PublishPayloadFuncOnTopicIfSubscribed(topic, payloadFunc)

	case *iotago.AccountOutput:
		accountID := o.AccountID
		if accountID.Empty() {
			// Use implicit AccountID
			accountID = iotago.AccountIDFromOutputID(outputID)
		}
		topic := strings.ReplaceAll(topicAccountOutputs, parameterAccountID, accountID.String())
		s.PublishPayloadFuncOnTopicIfSubscribed(topic, payloadFunc)

	case *iotago.FoundryOutput:
		foundryID, err := o.ID()
		if err != nil {
			return
		}
		topic := strings.ReplaceAll(topicFoundryOutputs, parameterFoundryID, foundryID.String())
		s.PublishPayloadFuncOnTopicIfSubscribed(topic, payloadFunc)

	default:
	}
}

func (s *Server) PublishOutput(ctx context.Context, output *inx.LedgerOutput, publishOnAllTopics bool) {
	api := s.NodeBridge.APIProvider().CurrentAPI()
	iotaOutput, err := output.UnwrapOutput(api)
	if err != nil {
		return
	}

	var payload *outputPayload
	payloadFunc := func() interface{} {
		if payload == nil {
			payload = payloadForOutput(api, output, iotaOutput)
		}

		return payload
	}

	outputID := output.GetOutputId().Unwrap()
	outputsTopic := strings.ReplaceAll(topicOutputs, parameterOutputID, outputID.ToHex())
	s.PublishPayloadFuncOnTopicIfSubscribed(outputsTopic, payloadFunc)

	if publishOnAllTopics {
		// If this is the first output in a transaction (index 0), then check if someone is observing the transaction that generated this output
		if outputID.Index() == 0 {
			ctxFetch, cancelFetch := context.WithTimeout(ctx, fetchTimeout)
			defer cancelFetch()

			transactionID := outputID.TransactionID()
			if s.hasSubscriberForTransactionIncludedBlock(transactionID) {
				s.fetchAndPublishTransactionInclusionWithBlock(ctxFetch, transactionID, output.GetBlockId().Unwrap())
			}
		}

		s.PublishOnOutputChainTopics(outputID, iotaOutput, payloadFunc)
		s.PublishOnUnlockConditionTopics(topicOutputsByUnlockConditionAndAddress, iotaOutput, payloadFunc)
	}
}

func (s *Server) PublishOutputMetadata(outputID iotago.OutputID, metadata *inx.OutputMetadata) {
	outputMetadataTopic := strings.ReplaceAll(topicOutputsMetadata, parameterOutputID, outputID.ToHex())

	response := &outputMetadataPayload{
		BlockID:              metadata.GetBlockId().Unwrap().ToHex(),
		TransactionID:        metadata.GetTransactionId().Unwrap().ToHex(),
		OutputIndex:          uint16(metadata.GetOutputIndex()),
		Spent:                metadata.GetIsSpent(),
		CommitmentIDSpent:    metadata.GetCommitmentIdSpent().Unwrap().ToHex(),
		TransactionIDSpent:   metadata.GetTransactionIdSpent().Unwrap().ToHex(),
		IncludedCommitmentID: metadata.GetCommitmentIdIncluded().Unwrap().ToHex(),
	}

	// Serialize here instead of using publishOnTopic to avoid double JSON marshaling
	jsonPayload, err := json.Marshal(response)
	if err != nil {
		return
	}

	s.sendMessageOnTopic(outputMetadataTopic, jsonPayload)
}

func (s *Server) PublishSpent(spent *inx.LedgerSpent) {
	api := s.NodeBridge.APIProvider().CurrentAPI()
	iotaOutput, err := spent.GetOutput().UnwrapOutput(api)
	if err != nil {
		return
	}

	var payload *outputPayload
	payloadFunc := func() interface{} {
		if payload == nil {
			payload = payloadForSpent(api, spent, iotaOutput)
		}

		return payload
	}

	outputsTopic := strings.ReplaceAll(topicOutputs, parameterOutputID, spent.GetOutput().GetOutputId().Unwrap().ToHex())
	s.PublishPayloadFuncOnTopicIfSubscribed(outputsTopic, payloadFunc)

	s.PublishOnUnlockConditionTopics(topicSpentOutputsByUnlockConditionAndAddress, iotaOutput, payloadFunc)
}

func blockIDFromBlockMetadataTopic(topic string) iotago.BlockID {
	if strings.HasPrefix(topic, "block-metadata/") && !strings.HasSuffix(topic, "/referenced") {
		blockIDHex := strings.Replace(topic, "block-metadata/", "", 1)
		blockID, err := iotago.SlotIdentifierFromHexString(blockIDHex)
		if err != nil {
			return iotago.EmptyBlockID()
		}

		return blockID
	}

	return iotago.EmptyBlockID()
}

func transactionIDFromTransactionsIncludedBlockTopic(topic string) iotago.TransactionID {
	if strings.HasPrefix(topic, "transactions/") && strings.HasSuffix(topic, "/included-block") {
		transactionIDHex := strings.Replace(topic, "transactions/", "", 1)
		transactionIDHex = strings.Replace(transactionIDHex, "/included-block", "", 1)

		transactionID, err := iotago.IdentifierFromHexString(transactionIDHex)
		if err != nil || len(transactionID) != iotago.TransactionIDLength {
			return emptyTransactionID
		}

		return transactionID
	}

	return emptyTransactionID
}

func outputIDFromOutputsTopic(topic string) iotago.OutputID {
	if strings.HasPrefix(topic, "outputs/") && !strings.HasPrefix(topic, "outputs/unlock") {
		outputIDHex := strings.Replace(topic, "outputs/", "", 1)
		outputID, err := iotago.OutputIDFromHex(outputIDHex)
		if err != nil {
			return emptyOutputID
		}

		return outputID
	}

	return emptyOutputID
}

func outputIDFromOutputMetadataTopic(topic string) iotago.OutputID {
	if strings.HasPrefix(topic, "output-metadata/") {
		outputIDHex := strings.Replace(topic, "output-metadata/", "", 1)
		outputID, err := iotago.OutputIDFromHex(outputIDHex)
		if err != nil {
			return emptyOutputID
		}

		return outputID
	}

	return emptyOutputID
}
