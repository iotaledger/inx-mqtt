package mqtt

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/iotaledger/hive.go/serializer/v2"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
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

func (s *Server) PublishCommitmentOnTopic(topic string, ms *nodebridge.Commitment) {
	if ms == nil || ms.Commitment == nil {
		return
	}

	s.PublishOnTopicIfSubscribed(topic, &commitemntInfoPayload{
		CommitmentID:        ms.CommitmentID.ToHex(),
		CommitmentSlotIndex: uint64(ms.Commitment.Index),
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

	block, err := blk.UnwrapBlock(apiForVersion, nil)
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

func hexEncodedBlockIDsFromINXBlockIDs(s []*inx.BlockId) []string {
	results := make([]string, len(s))
	for i, blkID := range s {
		results[i] = blkID.Unwrap().ToHex()
	}

	return results
}

func (s *Server) PublishBlockMetadata(metadata *inx.BlockMetadata) {
	blockID := metadata.GetBlockId().Unwrap().ToHex()
	singleBlockTopic := strings.ReplaceAll(topicBlockMetadata, parameterBlockID, blockID)
	hasSingleBlockTopicSubscriber := s.MQTTBroker.HasSubscribers(singleBlockTopic)
	hasAllBlocksTopicSubscriber := s.MQTTBroker.HasSubscribers(topicBlockMetadataReferenced)

	if !hasSingleBlockTopicSubscriber && !hasAllBlocksTopicSubscriber {
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
	if hasAllBlocksTopicSubscriber {
		s.sendMessageOnTopic(topicBlockMetadataReferenced, jsonPayload)
	}
}

func payloadForOutput(api iotago.API, ledgerIndex uint32, output *inx.LedgerOutput, iotaOutput iotago.Output) *outputPayload {
	rawOutputJSON, err := api.JSONEncode(iotaOutput)
	if err != nil {
		return nil
	}

	rawRawOutputJSON := json.RawMessage(rawOutputJSON)

	return &outputPayload{
		RawOutput: &rawRawOutputJSON,
	}
}

func payloadForSpent(api iotago.API, ledgerIndex uint32, spent *inx.LedgerSpent, iotaOutput iotago.Output) *outputPayload {
	payload := payloadForOutput(api, ledgerIndex, spent.GetOutput(), iotaOutput)
	if payload != nil {
		payload.Metadata.Spent = true
		payload.Metadata.MilestoneIndexSpent = spent.GetMilestoneIndexSpent()
		payload.Metadata.TransactionIDSpent = spent.UnwrapTransactionIDSpent().ToHex()
		payload.Metadata.MilestoneTimestampSpent = spent.GetMilestoneTimestampSpent()
	}

	return payload
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

	immutableAlias := unlockConditions.ImmutableAlias()
	if immutableAlias != nil {
		addr := immutableAlias.Address.Bech32(s.NodeBridge.ProtocolParameters().Bech32HRP)
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
		topic := strings.ReplaceAll(topicAliasOutputs, parameterAccountID, accountID.String())
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

func (s *Server) PublishOutput(ctx context.Context, ledgerIndex uint32, output *inx.LedgerOutput, publishOnAllTopics bool) {
	// get api by verson or ledgerIndex?
	api := s.NodeBridge.APIProvider().CurrentAPI()
	iotaOutput, err := output.UnwrapOutput(api, nil)
	if err != nil {
		return
	}

	var payload *outputPayload
	payloadFunc := func() interface{} {
		if payload == nil {
			payload = payloadForOutput(api, ledgerIndex, output, iotaOutput)
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

func (s *Server) PublishSpent(ledgerIndex uint32, spent *inx.LedgerSpent) {

	iotaOutput, err := spent.GetOutput().UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return
	}

	var payload *outputPayload
	payloadFunc := func() interface{} {
		if payload == nil {
			payload = payloadForSpent(ledgerIndex, spent, iotaOutput)
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
		blockID, err := iotago.BlockIDFromHexString(blockIDHex)
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

		decoded, err := iotago.DecodeHex(transactionIDHex)
		if err != nil || len(decoded) != iotago.TransactionIDLength {
			return emptyTransactionID
		}
		transactionID := iotago.TransactionID{}
		copy(transactionID[:], decoded)

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
