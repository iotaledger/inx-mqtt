package mqtt

import (
	"context"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/iotaledger/hive.go/app/shutdown"
	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/web/subscriptionmanager"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	"github.com/iotaledger/inx-mqtt/pkg/mqtt"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v4"
)

const (
	grpcListenToBlocks               = "INX.ListenToBlocks"
	grpcListenToAcceptedBlocks       = "INX.ListenToAcceptedBlocks"
	grpcListenToConfirmedBlocks      = "INX.ListenToConfirmedBlocks"
	grpcListenToAcceptedTransactions = "INX.ListenToAcceptedTransactions"
	grpcListenToLedgerUpdates        = "INX.ListenToLedgerUpdates"
)

const (
	fetchTimeout = 5 * time.Second
)

type topicSubcription struct {
	Count      int
	CancelFunc func()
	Identifier int
}

type Server struct {
	*logger.WrappedLogger

	MQTTBroker      *mqtt.Broker
	NodeBridge      *nodebridge.NodeBridge
	shutdownHandler *shutdown.ShutdownHandler
	brokerOptions   *mqtt.BrokerOptions

	grpcSubscriptionsLock sync.Mutex
	grpcSubscriptions     map[string]*topicSubcription
}

func NewServer(log *logger.Logger,
	bridge *nodebridge.NodeBridge,
	shutdownHandler *shutdown.ShutdownHandler,
	brokerOpts ...mqtt.BrokerOption) (*Server, error) {
	opts := &mqtt.BrokerOptions{}
	opts.ApplyOnDefault(brokerOpts...)

	s := &Server{
		WrappedLogger:     logger.NewWrappedLogger(log),
		NodeBridge:        bridge,
		shutdownHandler:   shutdownHandler,
		brokerOptions:     opts,
		grpcSubscriptions: make(map[string]*topicSubcription),
	}

	return s, nil
}

func (s *Server) Run(ctx context.Context) {
	broker, err := mqtt.NewBroker(s.brokerOptions)
	if err != nil {
		s.LogErrorfAndExit("failed to create MQTT broker: %s", err.Error())
	}

	// register broker events
	unhookBrokerEvents := lo.Batch(
		broker.Events().ClientConnected.Hook(func(event *subscriptionmanager.ClientEvent[string]) {
			s.onClientConnect(event.ClientID)
		}).Unhook,
		broker.Events().ClientDisconnected.Hook(func(event *subscriptionmanager.ClientEvent[string]) {
			s.onClientDisconnect(event.ClientID)
		}).Unhook,
		broker.Events().TopicSubscribed.Hook(func(event *subscriptionmanager.ClientTopicEvent[string, string]) {
			s.onSubscribeTopic(ctx, event.ClientID, event.Topic)
		}).Unhook,
		broker.Events().TopicUnsubscribed.Hook(func(event *subscriptionmanager.ClientTopicEvent[string, string]) {
			s.onUnsubscribeTopic(event.ClientID, event.Topic)
		}).Unhook,
	)
	s.MQTTBroker = broker

	if err := broker.Start(); err != nil {
		s.LogErrorfAndExit("failed to start MQTT broker: %s", err.Error())
	}

	if s.brokerOptions.WebsocketEnabled {
		ctxRegister, cancelRegister := context.WithTimeout(ctx, 5*time.Second)

		s.LogInfo("Registering API route ...")

		advertisedAddress := s.brokerOptions.WebsocketBindAddress
		if s.brokerOptions.WebsocketAdvertiseAddress != "" {
			advertisedAddress = s.brokerOptions.WebsocketAdvertiseAddress
		}

		if err := deps.NodeBridge.RegisterAPIRoute(ctxRegister, APIRoute, advertisedAddress, ""); err != nil {
			s.LogErrorfAndExit("failed to register API route via INX: %s", err.Error())
		}
		s.LogInfo("Registering API route ... done")
		cancelRegister()
	}

	// register node bridge events
	unhookNodeBridgeEvents := lo.Batch(
		s.NodeBridge.Events.LatestCommitmentChanged.Hook(func(c *nodebridge.Commitment) {
			if err := s.publishCommitmentOnTopicIfSubscribed(topicCommitmentsLatest, func() (*iotago.Commitment, error) { return c.Commitment, nil }); err != nil {
				s.LogWarnf("failed to publish latest commitment: %s", err.Error())
			}
		}).Unhook,

		s.NodeBridge.Events.LatestFinalizedCommitmentChanged.Hook(func(c *nodebridge.Commitment) {
			if err := s.publishCommitmentOnTopicIfSubscribed(topicCommitmentsFinalized, func() (*iotago.Commitment, error) { return c.Commitment, nil }); err != nil {
				s.LogWarnf("failed to publish latest finalized commitment: %s", err.Error())
			}
		}).Unhook,
	)

	s.LogInfo("Starting MQTT Broker ... done")
	<-ctx.Done()

	s.LogInfo("Stopping MQTT Broker ...")
	unhookBrokerEvents()
	unhookNodeBridgeEvents()

	if s.brokerOptions.WebsocketEnabled {
		ctxUnregister, cancelUnregister := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelUnregister()

		s.LogInfo("Removing API route ...")
		//nolint:contextcheck // false positive
		if err := deps.NodeBridge.UnregisterAPIRoute(ctxUnregister, APIRoute); err != nil {
			s.LogErrorf("failed to remove API route via INX: %s", err.Error())
		}
		cancelUnregister()
	}

	if err := s.MQTTBroker.Stop(); err != nil {
		s.LogErrorf("failed to stop MQTT broker: %s", err.Error())
	}

	s.LogInfo("Stopping MQTT Broker ...done")
}

func (s *Server) onClientConnect(clientID string) {
	s.LogDebugf("onClientConnect %s", clientID)
}

func (s *Server) onClientDisconnect(clientID string) {
	s.LogDebugf("onClientDisconnect %s", clientID)
}

func (s *Server) onSubscribeTopic(ctx context.Context, clientID string, topic string) {
	s.LogDebugf("client %s subscribed to %s", clientID, topic)

	switch topic {
	case topicCommitmentsLatest:
		// we don't need to subscribe here, because this is handled by the node bridge events
		// but we need to publish the latest payload once to the new subscriber
		go s.fetchAndPublishLatestCommitmentTopic()

	case topicCommitmentsFinalized:
		// we don't need to subscribe here, because this is handled by the node bridge events
		// but we need to publish the latest payload once to the new subscriber
		go s.fetchAndPublishFinalizedCommitmentTopic()

	case topicBlocks,
		topicBlocksValidation,
		topicBlocksBasic,
		topicBlocksBasicTransaction,
		topicBlocksBasicTransactionTaggedData,
		topicBlocksBasicTaggedData:
		s.startListenIfNeeded(ctx, grpcListenToBlocks, s.listenToBlocks)

	case topicBlockMetadataAccepted:
		s.startListenIfNeeded(ctx, grpcListenToAcceptedBlocks, s.listenToAcceptedBlocksMetadata)

	case topicBlockMetadataConfirmed:
		s.startListenIfNeeded(ctx, grpcListenToConfirmedBlocks, s.listenToConfirmedBlocksMetadata)

	default:
		switch {
		case strings.HasPrefix(topic, "blocks/basic/") && strings.Contains(topic, "tagged-data/"):
			// topicBlocksBasicTransactionTaggedDataTag
			// topicBlocksBasicTaggedDataTag
			s.startListenIfNeeded(ctx, grpcListenToBlocks, s.listenToBlocks)

		case strings.HasPrefix(topic, "block-metadata/"):
			// topicBlockMetadata
			// HINT: it can't be topicBlockMetadataAccepted or topicBlockMetadataConfirmed because they are handled above
			// so it must be a blockID
			if blockID := blockIDFromBlockMetadataTopic(topic); !blockID.Empty() {
				// start listening to accepted and confirmed blocks if not already done to get state updates for that blockID
				s.startListenIfNeeded(ctx, grpcListenToAcceptedBlocks, s.listenToAcceptedBlocksMetadata)
				s.startListenIfNeeded(ctx, grpcListenToConfirmedBlocks, s.listenToConfirmedBlocksMetadata)

				go s.fetchAndPublishBlockMetadata(ctx, blockID)
			}

		case strings.HasPrefix(topic, "outputs/") || strings.HasPrefix(topic, "transactions/"):
			// topicOutputs
			// topicAccountOutputs
			// topicAnchorOutputs
			// topicFoundryOutputs
			// topicNFTOutputs
			// topicOutputsByUnlockConditionAndAddress
			// topicSpentOutputsByUnlockConditionAndAddress
			// topicTransactionsIncludedBlock
			s.startListenIfNeeded(ctx, grpcListenToAcceptedTransactions, s.listenToAcceptedTransactions)
			s.startListenIfNeeded(ctx, grpcListenToLedgerUpdates, s.listenToLedgerUpdates)

			if transactionID := transactionIDFromTransactionsIncludedBlockTopic(topic); transactionID != iotago.EmptyTransactionID {
				go s.fetchAndPublishTransactionInclusion(ctx, transactionID)
			}
			if outputID := outputIDFromOutputsTopic(topic); outputID != iotago.EmptyOutputID {
				go s.fetchAndPublishOutput(ctx, outputID)
			}
		}
	}
}

func (s *Server) onUnsubscribeTopic(clientID string, topic string) {
	s.LogDebugf("client %s unsubscribed from %s", clientID, topic)

	switch topic {

	case topicCommitmentsLatest,
		topicCommitmentsFinalized:
		// we don't need to unsubscribe here, because this is handled by the node bridge events anyway.

	case topicBlocks,
		topicBlocksValidation,
		topicBlocksBasic,
		topicBlocksBasicTransaction,
		topicBlocksBasicTransactionTaggedData,
		topicBlocksBasicTaggedData:
		s.stopListenIfNeeded(grpcListenToBlocks)

	case topicBlockMetadataAccepted:
		s.stopListenIfNeeded(grpcListenToAcceptedBlocks)

	case topicBlockMetadataConfirmed:
		s.stopListenIfNeeded(grpcListenToConfirmedBlocks)

	default:
		switch {
		case strings.HasPrefix(topic, "blocks/basic/") && strings.Contains(topic, "tagged-data/"):
			// topicBlocksBasicTransactionTaggedDataTag
			// topicBlocksBasicTaggedDataTag
			s.stopListenIfNeeded(grpcListenToBlocks)

		case strings.HasPrefix(topic, "block-metadata/"):
			// topicBlockMetadata
			// it can't be topicBlockMetadataAccepted or topicBlockMetadataConfirmed because they are handled above
			s.stopListenIfNeeded(grpcListenToAcceptedBlocks)
			s.stopListenIfNeeded(grpcListenToConfirmedBlocks)

		case strings.HasPrefix(topic, "outputs/") || strings.HasPrefix(topic, "transactions/"):
			// topicOutputs
			// topicAccountOutputs
			// topicAnchorOutputs
			// topicFoundryOutputs
			// topicNFTOutputs
			// topicOutputsByUnlockConditionAndAddress
			// topicSpentOutputsByUnlockConditionAndAddress
			// topicTransactionsIncludedBlock
			s.stopListenIfNeeded(grpcListenToAcceptedTransactions)
			s.stopListenIfNeeded(grpcListenToLedgerUpdates)
		}
	}
}

func (s *Server) stopListenIfNeeded(grpcCall string) {
	s.grpcSubscriptionsLock.Lock()
	defer s.grpcSubscriptionsLock.Unlock()

	sub, ok := s.grpcSubscriptions[grpcCall]
	if ok {
		// subscription found
		// decrease amount of subscribers
		sub.Count--

		if sub.Count == 0 {
			// => no more subscribers => stop listening
			sub.CancelFunc()
			delete(s.grpcSubscriptions, grpcCall)
		}
	}
}

func (s *Server) startListenIfNeeded(ctx context.Context, grpcCall string, listenFunc func(context.Context) error) {
	s.grpcSubscriptionsLock.Lock()
	defer s.grpcSubscriptionsLock.Unlock()

	sub, ok := s.grpcSubscriptions[grpcCall]
	if ok {
		// subscription already exists
		// => increase count to track subscribers
		sub.Count++

		return
	}

	ctxCancel, cancel := context.WithCancel(ctx)

	//nolint:gosec // we do not care about weak random numbers here
	subscriptionIdentifier := rand.Int()
	s.grpcSubscriptions[grpcCall] = &topicSubcription{
		Count:      1,
		CancelFunc: cancel,
		Identifier: subscriptionIdentifier,
	}
	go func() {
		s.LogInfof("Listen to %s", grpcCall)
		err := listenFunc(ctxCancel)
		if err != nil && !errors.Is(err, context.Canceled) {
			s.LogErrorf("Finished listen to %s with error: %s", grpcCall, err.Error())
			if status.Code(err) == codes.Unavailable {
				s.shutdownHandler.SelfShutdown("INX became unavailable", true)
			}
		} else {
			s.LogInfof("Finished listen to %s", grpcCall)
		}
		s.grpcSubscriptionsLock.Lock()
		sub, ok := s.grpcSubscriptions[grpcCall]
		if ok && sub.Identifier == subscriptionIdentifier {
			// Only delete if it was not already replaced by a new one.
			delete(s.grpcSubscriptions, grpcCall)
		}
		s.grpcSubscriptionsLock.Unlock()
	}()
}

func (s *Server) listenToBlocks(ctx context.Context) error {
	stream, err := s.NodeBridge.Client().ListenToBlocks(ctx, &inx.NoParams{})
	if err != nil {
		return err
	}

	return nodebridge.ListenToStream(ctx, stream.Recv, func(block *inx.Block) error {
		if err := s.publishBlockIfSubscribed(block.GetBlock()); err != nil {
			s.LogErrorf("failed to publish block: %v", err)
		}

		// we don't return an error here, because we want to continue listening even if publishing fails once
		return nil
	})
}

func (s *Server) listenToAcceptedBlocksMetadata(ctx context.Context) error {
	stream, err := s.NodeBridge.Client().ListenToAcceptedBlocks(ctx, &inx.NoParams{})
	if err != nil {
		return err
	}

	return nodebridge.ListenToStream(ctx, stream.Recv, func(blockMetadata *inx.BlockMetadata) error {
		if err := s.publishBlockMetadataOnTopicIfSubscribed(func() (*inx.BlockMetadata, error) { return blockMetadata, nil }, topicBlockMetadataAccepted); err != nil {
			s.LogErrorf("failed to publish accepted block metadata: %v", err)
		}

		// we don't return an error here, because we want to continue listening even if publishing fails once
		return nil
	})
}

func (s *Server) listenToConfirmedBlocksMetadata(ctx context.Context) error {
	stream, err := s.NodeBridge.Client().ListenToConfirmedBlocks(ctx, &inx.NoParams{})
	if err != nil {
		return err
	}

	return nodebridge.ListenToStream(ctx, stream.Recv, func(blockMetadata *inx.BlockMetadata) error {
		if err := s.publishBlockMetadataOnTopicIfSubscribed(func() (*inx.BlockMetadata, error) { return blockMetadata, nil }, topicBlockMetadataConfirmed); err != nil {
			s.LogErrorf("failed to publish confirmed block metadata: %v", err)
		}

		// we don't return an error here, because we want to continue listening even if publishing fails once
		return nil
	})
}

func (s *Server) listenToAcceptedTransactions(ctx context.Context) error {
	return s.NodeBridge.ListenToAcceptedTransactions(ctx, func(payload *nodebridge.AcceptedTransaction) error {
		for _, consumed := range payload.Consumed {
			if err := s.publishSpentIfSubscribed(ctx, consumed, true); err != nil {
				s.LogErrorf("failed to publish spent output in listen to accepted transaction update: %v", err)
			}
		}

		for _, created := range payload.Created {
			if err := s.publishOutputIfSubscribed(ctx, created, true); err != nil {
				s.LogErrorf("failed to publish created output in listen to accepted transaction update: %v", err)
			}
		}

		// we don't return an error here, because we want to continue listening even if publishing fails once
		return nil
	})
}

func (s *Server) listenToLedgerUpdates(ctx context.Context) error {
	return s.NodeBridge.ListenToLedgerUpdates(ctx, 0, 0, func(payload *nodebridge.LedgerUpdate) error {
		for _, consumed := range payload.Consumed {
			if err := s.publishSpentIfSubscribed(ctx, consumed, true); err != nil {
				s.LogErrorf("failed to publish spent output in ledger update: %v", err)
			}
		}

		for _, created := range payload.Created {
			if err := s.publishOutputIfSubscribed(ctx, created, true); err != nil {
				s.LogErrorf("failed to publish created output in ledger update: %v", err)
			}
		}

		// we don't return an error here, because we want to continue listening even if publishing fails once
		return nil
	})
}

func (s *Server) fetchAndPublishLatestCommitmentTopic() {
	if err := s.publishCommitmentOnTopicIfSubscribed(topicCommitmentsLatest,
		func() (*iotago.Commitment, error) {
			latestCommitment := s.NodeBridge.LatestCommitment()
			if latestCommitment == nil {
				return nil, ierrors.New("failed to retrieve latest commitment")
			}

			return latestCommitment.Commitment, nil
		},
	); err != nil {
		s.LogErrorf("failed to publish latest commitment: %v", err)
	}
}

func (s *Server) fetchAndPublishFinalizedCommitmentTopic() {
	if err := s.publishCommitmentOnTopicIfSubscribed(topicCommitmentsFinalized,
		func() (*iotago.Commitment, error) {
			latestFinalizedCommitment := s.NodeBridge.LatestFinalizedCommitment()
			if latestFinalizedCommitment == nil {
				return nil, ierrors.New("failed to retrieve latest finalized commitment")
			}

			return latestFinalizedCommitment.Commitment, nil
		},
	); err != nil {
		s.LogErrorf("failed to publish latest finalized commitment: %v", err)
	}
}

func (s *Server) fetchAndPublishBlockMetadata(ctx context.Context, blockID iotago.BlockID) {
	if err := s.publishBlockMetadataOnTopicIfSubscribed(func() (*inx.BlockMetadata, error) {
		resp, err := s.NodeBridge.Client().ReadBlockMetadata(ctx, inx.NewBlockId(blockID))
		if err != nil {
			return nil, ierrors.Wrapf(err, "failed to retrieve block metadata %s", blockID.ToHex())
		}

		return resp, nil
	}, getTopicBlockMetadata(blockID)); err != nil {
		s.LogErrorf("failed to publish block metadata %s: %v", blockID.ToHex(), err)
	}
}

func (s *Server) fetchAndPublishOutput(ctx context.Context, outputID iotago.OutputID) {
	// we need to fetch the output to figure out which topics we need to publish on
	resp, err := s.NodeBridge.Client().ReadOutput(ctx, inx.NewOutputId(outputID))
	if err != nil {
		s.LogErrorf("failed to retrieve output %s: %v", outputID.ToHex(), err)
		return
	}

	if resp.GetSpent() != nil {
		if err := s.publishSpentIfSubscribed(ctx, resp.GetSpent(), false, resp.GetLatestCommitmentId().Unwrap()); err != nil {
			s.LogErrorf("failed to publish spent output %s: %v", outputID.ToHex(), err)
		}
	} else {
		if err := s.publishOutputIfSubscribed(ctx, resp.GetOutput(), false, resp.GetLatestCommitmentId().Unwrap()); err != nil {
			s.LogErrorf("failed to publish output %s: %v", outputID.ToHex(), err)
		}
	}
}

func (s *Server) fetchAndPublishTransactionInclusion(ctx context.Context, transactionID iotago.TransactionID) {

	var blockID iotago.BlockID
	blockIDFunc := func() (iotago.BlockID, error) {
		if blockID.Empty() {
			// get the output and then the blockID of the transaction that created the output
			outputID := iotago.OutputID{}
			copy(outputID[:], transactionID[:])

			ctxFetch, cancelFetch := context.WithTimeout(ctx, fetchTimeout)
			defer cancelFetch()

			resp, err := s.NodeBridge.Client().ReadOutput(ctxFetch, inx.NewOutputId(outputID))
			if err != nil {
				return iotago.EmptyBlockID, ierrors.Wrapf(err, "failed to retrieve output of transaction %s", transactionID.ToHex())
			}

			inxOutput := resp.GetOutput()
			if resp.GetSpent() != nil {
				inxOutput = resp.GetSpent().GetOutput()
			}

			blockID = inxOutput.GetBlockId().Unwrap()
		}

		return blockID, nil
	}

	s.fetchAndPublishTransactionInclusionWithBlock(ctx, transactionID, blockIDFunc)
}

func (s *Server) fetchAndPublishTransactionInclusionWithBlock(ctx context.Context, transactionID iotago.TransactionID, blockIDFunc func() (iotago.BlockID, error)) {
	ctxFetch, cancelFetch := context.WithTimeout(ctx, fetchTimeout)
	defer cancelFetch()

	var block *iotago.Block
	blockFunc := func() (*iotago.Block, error) {
		blockID, err := blockIDFunc()
		if err != nil {
			return nil, err
		}

		resp, err := s.NodeBridge.Client().ReadBlock(ctxFetch, inx.NewBlockId(blockID))
		if err != nil {
			s.LogErrorf("failed to retrieve block %s :%v", blockID.ToHex(), err)
			return nil, err
		}

		block, err = resp.UnwrapBlock(s.NodeBridge.APIProvider())
		if err != nil {
			return nil, err
		}

		return block, nil
	}

	if err := s.publishPayloadOnTopicsIfSubscribed(
		func() (iotago.API, error) {
			block, err := blockFunc()
			if err != nil {
				return nil, err
			}

			return block.API, nil
		},
		func() (any, error) {
			return blockFunc()
		},
		getTransactionsIncludedBlockTopic(transactionID),
	); err != nil {
		s.LogErrorf("failed to publish transaction inclusion %s: %v", transactionID.ToHex(), err)
	}
}
