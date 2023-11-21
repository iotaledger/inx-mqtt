package mqtt

import (
	"context"

	"github.com/iotaledger/hive.go/ierrors"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v4"
	"github.com/iotaledger/iota.go/v4/nodeclient/apimodels"
)

func (s *Server) sendMessageOnTopic(topic string, payload []byte) {
	if err := s.MQTTBroker.Send(topic, payload); err != nil {
		s.LogWarnf("failed to send message on topic %s, error: %s", topic, err.Error())
	}
}

func (s *Server) publishWithPayloadFuncOnTopicsIfSubscribed(payloadFunc func() ([]byte, error), topics ...string) error {
	var payload []byte

	for _, topic := range topics {
		if !s.MQTTBroker.HasSubscribers(topic) {
			continue
		}

		if payload == nil {
			// payload is not yet set, so we need to call the payloadFunc
			var err error
			payload, err = payloadFunc()
			if err != nil {
				return err
			}
		}

		s.sendMessageOnTopic(topic, payload)
	}

	return nil
}

func addTopicsPrefix(topics []string, prefix string) []string {
	rawTopics := make([]string, len(topics))

	for i, topic := range topics {
		rawTopics[i] = topic + prefix
	}

	return rawTopics
}

func (s *Server) publishWithPayloadFuncsOnTopicsIfSubscribed(jsonPayloadFunc func() ([]byte, error), rawPayloadFunc func() ([]byte, error), topics ...string) error {
	// publish JSON payload
	if err := s.publishWithPayloadFuncOnTopicsIfSubscribed(func() ([]byte, error) {
		return jsonPayloadFunc()
	}, topics...); err != nil {
		return err
	}

	// publish raw payload
	if err := s.publishWithPayloadFuncOnTopicsIfSubscribed(func() ([]byte, error) {
		return rawPayloadFunc()
	}, addTopicsPrefix(topics, "/raw")...); err != nil {
		return err
	}

	return nil
}

func (s *Server) publishPayloadOnTopicsIfSubscribed(apiFunc func() (iotago.API, error), payloadFunc func() (any, error), topics ...string) error {
	return s.publishWithPayloadFuncsOnTopicsIfSubscribed(func() ([]byte, error) {
		api, err := apiFunc()
		if err != nil {
			return nil, err
		}

		payload, err := payloadFunc()
		if err != nil {
			return nil, err
		}

		return api.JSONEncode(payload)
	}, func() ([]byte, error) {
		api, err := apiFunc()
		if err != nil {
			return nil, err
		}

		payload, err := payloadFunc()
		if err != nil {
			return nil, err
		}

		return api.Encode(payload)
	}, topics...)
}

func (s *Server) publishCommitmentOnTopicIfSubscribed(topic string, commitmentFunc func() (*iotago.Commitment, error)) error {
	return s.publishPayloadOnTopicsIfSubscribed(
		func() (iotago.API, error) {
			commitment, err := commitmentFunc()
			if err != nil {
				return nil, err
			}

			return s.NodeBridge.APIProvider().APIForVersion(commitment.ProtocolVersion)
		},
		func() (any, error) {
			return commitmentFunc()
		},
		topic,
	)
}

func (s *Server) blockTopicsForBasicBlock(basicBlockBody *iotago.BasicBlockBody) []string {
	blockTopics := []string{topicBlocksBasic}

	switch payload := basicBlockBody.Payload.(type) {
	case *iotago.SignedTransaction:
		blockTopics = append(blockTopics, topicBlocksBasicTransaction)

		//nolint:gocritic // the type switch is nicer here
		switch p := payload.Transaction.Payload.(type) {
		case *iotago.TaggedData:
			blockTopics = append(blockTopics, topicBlocksBasicTransactionTaggedData)
			if len(p.Tag) > 0 {
				blockTopics = append(blockTopics, getTopicBlocksBasicTransactionTaggedDataTag(p.Tag))
			}
		}

	case *iotago.TaggedData:
		blockTopics = append(blockTopics, topicBlocksBasicTaggedData)
		if len(payload.Tag) > 0 {
			blockTopics = append(blockTopics, getTopicBlocksBasicTaggedDataTag(payload.Tag))
		}
	}

	return blockTopics
}

func (s *Server) publishBlockIfSubscribed(blk *inx.RawBlock) error {
	// we need to unwrap the block to figure out which topics we need to publish on
	block, err := blk.UnwrapBlock(s.NodeBridge.APIProvider())
	if err != nil {
		return err
	}

	// always publish every block on the "blocks" topic
	blockTopics := []string{topicBlocks}

	switch blockBody := block.Body.(type) {
	case *iotago.BasicBlockBody:
		blockTopics = append(blockTopics, s.blockTopicsForBasicBlock(blockBody)...)
	case *iotago.ValidationBlockBody:
		blockTopics = append(blockTopics, topicBlocksValidation)
	default:
		s.LogWarnf("unknown block body type: %T", blockBody)
	}

	return s.publishWithPayloadFuncsOnTopicsIfSubscribed(func() ([]byte, error) {
		return block.API.JSONEncode(block)
	}, func() ([]byte, error) {
		return blk.GetData(), nil
	}, blockTopics...)
}

func (s *Server) publishBlockMetadataOnTopicIfSubscribed(metadataFunc func() (*inx.BlockMetadata, error), topic string) error {
	return s.publishPayloadOnTopicsIfSubscribed(
		func() (iotago.API, error) { return s.NodeBridge.APIProvider().CommittedAPI(), nil },
		func() (any, error) {
			metadata, err := metadataFunc()
			if err != nil {
				return nil, err
			}

			transactionStateString := ""

			//nolint:nosnakecase
			if metadata.GetTransactionState() != inx.BlockMetadata_TRANSACTION_STATE_NO_TRANSACTION {
				transactionStateString = apimodels.TransactionState(metadata.GetTransactionState()).String()
			}

			return &apimodels.BlockMetadataResponse{
				BlockID:                  metadata.GetBlockId().Unwrap(),
				BlockState:               apimodels.BlockState(metadata.GetBlockState()).String(),
				BlockFailureReason:       apimodels.BlockFailureReason(metadata.GetBlockFailureReason()),
				TransactionState:         transactionStateString,
				TransactionFailureReason: apimodels.TransactionFailureReason(metadata.GetTransactionFailureReason()),
			}, nil
		},
		topic,
	)
}

func (s *Server) publishOutputWithMetadataIfSubscribed(ctx context.Context, output *inx.LedgerOutput, payloadFunc func(api iotago.API, iotaOutput iotago.Output) (*apimodels.OutputWithMetadataResponse, error), publishOnAllTopics bool, publishTxInclusion bool) error {
	// we need to unwrap the output to figure out which topics we need to publish on
	api := s.NodeBridge.APIProvider().CommittedAPI()

	iotaOutput, err := output.UnwrapOutput(api)
	if err != nil {
		return err
	}

	topics := []string{getTopicOutput(output)}
	if publishOnAllTopics {
		outputID := output.GetOutputId().Unwrap()

		// If this is the first output in a transaction (index 0), then check if someone is observing the transaction that generated this output
		if publishTxInclusion && outputID.Index() == 0 {
			s.fetchAndPublishTransactionInclusionWithBlock(ctx,
				outputID.TransactionID(),
				func() (iotago.BlockID, error) {
					return output.GetBlockId().Unwrap(), nil
				},
			)
		}

		bech32HRP := s.NodeBridge.APIProvider().CommittedAPI().ProtocolParameters().Bech32HRP()
		topics = append(topics, getChainTopicsForOutput(outputID, iotaOutput, bech32HRP)...)
		topics = append(topics, getUnlockConditionTopicsForOutput(topicOutputsByUnlockConditionAndAddress, iotaOutput, bech32HRP)...)
	}

	var payload *apimodels.OutputWithMetadataResponse
	payloadFuncCached := func() (any, error) {
		if payload == nil {
			payload, err = payloadFunc(api, iotaOutput)
			if err != nil {
				return nil, err
			}
		}

		return payload, nil
	}

	return s.publishPayloadOnTopicsIfSubscribed(
		func() (iotago.API, error) {
			return api, nil
		},
		payloadFuncCached,
		topics...,
	)
}

func payloadForOutput(api iotago.API, output *inx.LedgerOutput, iotaOutput iotago.Output, latestCommitmentID iotago.CommitmentID) (*apimodels.OutputWithMetadataResponse, error) {
	outputID := output.GetOutputId().Unwrap()
	outputIDProof, err := output.UnwrapOutputIDProof(api)
	if err != nil {
		return nil, ierrors.Wrap(err, "failed to unwrap output ID proof")
	}

	return &apimodels.OutputWithMetadataResponse{
		Output:        iotaOutput,
		OutputIDProof: outputIDProof,
		Metadata: &apimodels.OutputMetadata{
			BlockID:              output.GetBlockId().Unwrap(),
			TransactionID:        outputID.TransactionID(),
			OutputIndex:          outputID.Index(),
			IsSpent:              false,
			CommitmentIDSpent:    iotago.EmptyCommitmentID,
			TransactionIDSpent:   iotago.EmptyTransactionID,
			IncludedCommitmentID: output.GetCommitmentIdIncluded().Unwrap(),
			LatestCommitmentID:   latestCommitmentID,
		},
	}, nil
}

func (s *Server) publishOutputIfSubscribed(ctx context.Context, output *inx.LedgerOutput, publishOnAllTopics bool, latestCommitmentID ...iotago.CommitmentID) error {
	return s.publishOutputWithMetadataIfSubscribed(ctx,
		output,
		func(api iotago.API, iotaOutput iotago.Output) (*apimodels.OutputWithMetadataResponse, error) {
			latestID := s.NodeBridge.LatestCommitment().CommitmentID
			if len(latestCommitmentID) > 0 {
				latestID = latestCommitmentID[0]
			}

			return payloadForOutput(api, output, iotaOutput, latestID)
		},
		publishOnAllTopics,
		true,
	)
}

func (s *Server) publishSpentIfSubscribed(ctx context.Context, spent *inx.LedgerSpent, publishOnAllTopics bool, latestCommitmentID ...iotago.CommitmentID) error {
	return s.publishOutputWithMetadataIfSubscribed(ctx,
		spent.GetOutput(),
		func(api iotago.API, iotaOutput iotago.Output) (*apimodels.OutputWithMetadataResponse, error) {
			latestID := s.NodeBridge.LatestCommitment().CommitmentID
			if len(latestCommitmentID) > 0 {
				latestID = latestCommitmentID[0]
			}

			payload, err := payloadForOutput(api, spent.GetOutput(), iotaOutput, latestID)
			if err != nil {
				return nil, err
			}

			payload.Metadata.IsSpent = true
			payload.Metadata.CommitmentIDSpent = spent.GetCommitmentIdSpent().Unwrap()
			payload.Metadata.TransactionIDSpent = spent.UnwrapTransactionIDSpent()

			return payload, nil
		},
		publishOnAllTopics,
		false,
	)
}
