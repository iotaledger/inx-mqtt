package mqtt

import (
	"context"

	"github.com/iotaledger/inx-app/pkg/nodebridge"
	iotago "github.com/iotaledger/iota.go/v4"
	"github.com/iotaledger/iota.go/v4/api"
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
	blockTopics := []string{api.EventAPITopicBlocksBasic}

	switch payload := basicBlockBody.Payload.(type) {
	case *iotago.SignedTransaction:
		blockTopics = append(blockTopics, api.EventAPITopicBlocksBasicTransaction)

		//nolint:gocritic // the type switch is nicer here
		switch p := payload.Transaction.Payload.(type) {
		case *iotago.TaggedData:
			blockTopics = append(blockTopics, api.EventAPITopicBlocksBasicTransactionTaggedData)
			if len(p.Tag) > 0 {
				blockTopics = append(blockTopics, GetTopicBlocksBasicTransactionTaggedDataTag(p.Tag))
			}
		}

	case *iotago.TaggedData:
		blockTopics = append(blockTopics, api.EventAPITopicBlocksBasicTaggedData)
		if len(payload.Tag) > 0 {
			blockTopics = append(blockTopics, GetTopicBlocksBasicTaggedDataTag(payload.Tag))
		}
	}

	return blockTopics
}

func (s *Server) publishBlockIfSubscribed(block *iotago.Block, rawData []byte) error {
	// always publish every block on the "blocks" topic
	blockTopics := []string{api.EventAPITopicBlocks}

	switch blockBody := block.Body.(type) {
	case *iotago.BasicBlockBody:
		blockTopics = append(blockTopics, s.blockTopicsForBasicBlock(blockBody)...)
	case *iotago.ValidationBlockBody:
		blockTopics = append(blockTopics, api.EventAPITopicBlocksValidation)
	default:
		s.LogWarnf("unknown block body type: %T", blockBody)
	}

	return s.publishWithPayloadFuncsOnTopicsIfSubscribed(func() ([]byte, error) {
		return block.API.JSONEncode(block)
	}, func() ([]byte, error) {
		return rawData, nil
	}, blockTopics...)
}

func (s *Server) publishBlockMetadataOnTopicsIfSubscribed(metadataFunc func() (*api.BlockMetadataResponse, error), topics ...string) error {
	return s.publishPayloadOnTopicsIfSubscribed(
		func() (iotago.API, error) { return s.NodeBridge.APIProvider().CommittedAPI(), nil },
		func() (any, error) {
			return metadataFunc()
		},
		topics...,
	)
}

func (s *Server) publishTransactionMetadataOnTopicsIfSubscribed(metadataFunc func() (*api.TransactionMetadataResponse, error), topics ...string) error {
	return s.publishPayloadOnTopicsIfSubscribed(
		func() (iotago.API, error) { return s.NodeBridge.APIProvider().CommittedAPI(), nil },
		func() (any, error) {
			return metadataFunc()
		},
		topics...,
	)
}

func (s *Server) publishOutputIfSubscribed(ctx context.Context, output *nodebridge.Output, publishOnAllTopics bool) error {
	topics := []string{GetTopicOutput(output.OutputID)}

	if publishOnAllTopics {
		// If this is the first output in a transaction (index 0), then check if someone is observing the transaction that generated this output
		if output.Metadata.Spent == nil && output.OutputID.Index() == 0 {
			s.fetchAndPublishTransactionInclusionBlockMetadataWithBlockID(ctx,
				output.OutputID.TransactionID(),
				func() (iotago.BlockID, error) {
					return output.Metadata.BlockID, nil
				},
			)
		}

		bech32HRP := s.NodeBridge.APIProvider().CommittedAPI().ProtocolParameters().Bech32HRP()
		topics = append(topics, GetChainTopicsForOutput(output.OutputID, output.Output, bech32HRP)...)
		topics = append(topics, GetUnlockConditionTopicsForOutput(api.EventAPITopicOutputsByUnlockConditionAndAddress, output.Output, bech32HRP)...)
	}

	var payload *api.OutputWithMetadataResponse
	payloadFuncCached := func() (any, error) {
		if payload == nil {
			payload = &api.OutputWithMetadataResponse{
				Output:        output.Output,
				OutputIDProof: output.OutputIDProof,
				Metadata:      output.Metadata,
			}
		}

		return payload, nil
	}

	return s.publishPayloadOnTopicsIfSubscribed(
		func() (iotago.API, error) {
			return s.NodeBridge.APIProvider().CommittedAPI(), nil
		},
		payloadFuncCached,
		topics...,
	)
}
