package mqtt

import (
	"context"
	"math/rand"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/iotaledger/hive.go/app/shutdown"
	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/log"
	"github.com/iotaledger/hive.go/web/subscriptionmanager"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	"github.com/iotaledger/inx-mqtt/pkg/broker"
	iotago "github.com/iotaledger/iota.go/v4"
	"github.com/iotaledger/iota.go/v4/api"
)

const (
	APIRoute = "mqtt/v2"
)

const (
	GrpcListenToBlocks               = "INX.ListenToBlocks"
	GrpcListenToAcceptedBlocks       = "INX.ListenToAcceptedBlocks"
	GrpcListenToConfirmedBlocks      = "INX.ListenToConfirmedBlocks"
	GrpcListenToAcceptedTransactions = "INX.ListenToAcceptedTransactions"
	GrpcListenToLedgerUpdates        = "INX.ListenToLedgerUpdates"
)

const (
	fetchTimeout = 5 * time.Second
)

type grpcSubcription struct {
	//nolint:containedctx
	context    context.Context
	cancelFunc func()
	count      int
	identifier int
}

type Server struct {
	log.Logger

	MQTTBroker      broker.Broker
	NodeBridge      nodebridge.NodeBridge
	shutdownHandler *shutdown.ShutdownHandler
	serverOptions   *Options

	grpcSubscriptionsLock sync.Mutex
	grpcSubscriptions     map[string]*grpcSubcription

	cleanupFunc func()
}

func NewServer(log log.Logger,
	bridge nodebridge.NodeBridge,
	broker broker.Broker,
	shutdownHandler *shutdown.ShutdownHandler,
	serverOpts ...Option) (*Server, error) {
	opts := &Options{}
	opts.ApplyOnDefault(serverOpts...)

	s := &Server{
		Logger:            log,
		NodeBridge:        bridge,
		shutdownHandler:   shutdownHandler,
		MQTTBroker:        broker,
		serverOptions:     opts,
		grpcSubscriptions: make(map[string]*grpcSubcription),
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) error {

	// register broker events
	unhookBrokerEvents := lo.Batch(
		s.MQTTBroker.Events().ClientConnected.Hook(func(event *subscriptionmanager.ClientEvent[string]) {
			s.onClientConnect(event.ClientID)
		}).Unhook,
		s.MQTTBroker.Events().ClientDisconnected.Hook(func(event *subscriptionmanager.ClientEvent[string]) {
			s.onClientDisconnect(event.ClientID)
		}).Unhook,
		s.MQTTBroker.Events().TopicSubscribed.Hook(func(event *subscriptionmanager.ClientTopicEvent[string, string]) {
			s.onSubscribeTopic(ctx, event.ClientID, event.Topic)
		}).Unhook,
		s.MQTTBroker.Events().TopicUnsubscribed.Hook(func(event *subscriptionmanager.ClientTopicEvent[string, string]) {
			s.onUnsubscribeTopic(event.ClientID, event.Topic)
		}).Unhook,
	)

	if err := s.MQTTBroker.Start(); err != nil {
		return ierrors.Wrap(err, "failed to start MQTT broker")
	}

	if s.serverOptions.WebsocketEnabled {
		ctxRegister, cancelRegister := context.WithTimeout(ctx, 5*time.Second)

		s.LogInfo("Registering API route ...")

		advertisedAddress := s.serverOptions.WebsocketBindAddress
		if s.serverOptions.WebsocketAdvertiseAddress != "" {
			advertisedAddress = s.serverOptions.WebsocketAdvertiseAddress
		}

		if err := s.NodeBridge.RegisterAPIRoute(ctxRegister, APIRoute, advertisedAddress, ""); err != nil {
			s.LogFatalf("failed to register API route via INX: %s", err.Error())
		}
		s.LogInfo("Registering API route ... done")
		cancelRegister()
	}

	// register node bridge events
	unhookNodeBridgeEvents := lo.Batch(
		s.NodeBridge.Events().LatestCommitmentChanged.Hook(func(c *nodebridge.Commitment) {
			if err := s.publishCommitmentOnTopicIfSubscribed(api.EventAPITopicCommitmentsLatest, func() (*iotago.Commitment, error) { return c.Commitment, nil }); err != nil {
				s.LogWarnf("failed to publish latest commitment: %s", err.Error())
			}
		}).Unhook,

		s.NodeBridge.Events().LatestFinalizedCommitmentChanged.Hook(func(c *nodebridge.Commitment) {
			if err := s.publishCommitmentOnTopicIfSubscribed(api.EventAPITopicCommitmentsFinalized, func() (*iotago.Commitment, error) { return c.Commitment, nil }); err != nil {
				s.LogWarnf("failed to publish latest finalized commitment: %s", err.Error())
			}
		}).Unhook,
	)

	s.cleanupFunc = lo.Batch(
		unhookBrokerEvents,
		unhookNodeBridgeEvents,
	)

	s.LogInfo("Starting MQTT Broker ... done")

	return nil
}

func (s *Server) Stop() error {
	s.LogInfo("Stopping MQTT Broker ...")

	if s.cleanupFunc != nil {
		s.cleanupFunc()
	}

	if s.serverOptions.WebsocketEnabled {
		ctxUnregister, cancelUnregister := context.WithTimeout(context.Background(), 5*time.Second)

		s.LogInfo("Removing API route ...")
		//nolint:contextcheck // false positive
		if err := s.NodeBridge.UnregisterAPIRoute(ctxUnregister, APIRoute); err != nil {
			s.LogErrorf("failed to remove API route via INX: %s", err.Error())
		}
		cancelUnregister()
	}

	if err := s.MQTTBroker.Stop(); err != nil {
		return ierrors.Wrap(err, "failed to stop MQTT broker")
	}

	s.LogInfo("Stopping MQTT Broker ...done")

	return nil
}

func (s *Server) Run(ctx context.Context) {
	if err := s.Start(ctx); err != nil {
		s.LogFatal(err.Error())
	}

	<-ctx.Done()

	//nolint:contextcheck // false positive
	if err := s.Stop(); err != nil {
		s.LogError(err.Error())
	}
}

func (s *Server) onClientConnect(clientID string) {
	s.LogDebugf("onClientConnect %s", clientID)
}

func (s *Server) onClientDisconnect(clientID string) {
	s.LogDebugf("onClientDisconnect %s", clientID)
}

func (s *Server) onSubscribeTopic(ctx context.Context, clientID string, topic string) {
	s.LogDebugf("client %s subscribed to %s", clientID, topic)

	// remove /raw suffix if present
	topic = strings.TrimSuffix(topic, "/raw")

	switch topic {
	case api.EventAPITopicCommitmentsLatest:
		// we don't need to subscribe here, because this is handled by the node bridge events
		// but we need to publish the latest payload once to the new subscriber
		go s.fetchAndPublishLatestCommitmentTopic()

	case api.EventAPITopicCommitmentsFinalized:
		// we don't need to subscribe here, because this is handled by the node bridge events
		// but we need to publish the latest payload once to the new subscriber
		go s.fetchAndPublishFinalizedCommitmentTopic()

	case api.EventAPITopicBlocks,
		api.EventAPITopicBlocksValidation,
		api.EventAPITopicBlocksBasic,
		api.EventAPITopicBlocksBasicTransaction,
		api.EventAPITopicBlocksBasicTransactionTaggedData,
		api.EventAPITopicBlocksBasicTaggedData:
		s.startListenIfNeeded(ctx, GrpcListenToBlocks, s.listenToBlocks)

	case api.EventAPITopicBlockMetadataAccepted:
		s.startListenIfNeeded(ctx, GrpcListenToAcceptedBlocks, s.listenToAcceptedBlocksMetadata)

	case api.EventAPITopicBlockMetadataConfirmed:
		s.startListenIfNeeded(ctx, GrpcListenToConfirmedBlocks, s.listenToConfirmedBlocksMetadata)

	default:
		switch {
		case strings.HasPrefix(topic, "blocks/basic/") && strings.Contains(topic, "tagged-data/"):
			// topicBlocksBasicTransactionTaggedDataTag
			// topicBlocksBasicTaggedDataTag
			s.startListenIfNeeded(ctx, GrpcListenToBlocks, s.listenToBlocks)

		case strings.HasPrefix(topic, "block-metadata/"):
			// topicBlockMetadata
			// HINT: it can't be topicBlockMetadataAccepted or topicBlockMetadataConfirmed because they are handled above
			// so it must be a blockID
			if blockID := BlockIDFromBlockMetadataTopic(topic); !blockID.Empty() {
				// start listening to accepted and confirmed blocks if not already done to get state updates for that blockID
				s.startListenIfNeeded(ctx, GrpcListenToAcceptedBlocks, s.listenToAcceptedBlocksMetadata)
				s.startListenIfNeeded(ctx, GrpcListenToConfirmedBlocks, s.listenToConfirmedBlocksMetadata)

				go s.fetchAndPublishBlockMetadata(ctx, blockID)
			}

		case strings.HasPrefix(topic, "outputs/") || strings.HasPrefix(topic, "transactions/") || strings.HasPrefix(topic, "transaction-metadata/"):
			// topicOutputs
			// topicAccountOutputs
			// topicAnchorOutputs
			// topicFoundryOutputs
			// topicNFTOutputs
			// topicOutputsByUnlockConditionAndAddress
			// topicSpentOutputsByUnlockConditionAndAddress
			// topicTransactionsIncludedBlockMetadata
			// topicTransactionMetadata
			s.startListenIfNeeded(ctx, GrpcListenToAcceptedTransactions, s.listenToAcceptedTransactions)
			s.startListenIfNeeded(ctx, GrpcListenToLedgerUpdates, s.listenToLedgerUpdates)

			if transactionID := TransactionIDFromTransactionsIncludedBlockMetadataTopic(topic); transactionID != iotago.EmptyTransactionID {
				go s.fetchAndPublishTransactionInclusionBlockMetadata(ctx, transactionID)
			}
			if transactionID := TransactionIDFromTransactionMetadataTopic(topic); transactionID != iotago.EmptyTransactionID {
				go s.fetchAndPublishTransactionMetadata(ctx, transactionID)
			}
			if outputID := OutputIDFromOutputsTopic(topic); outputID != iotago.EmptyOutputID {
				go s.fetchAndPublishOutput(ctx, outputID)
			}
		}
	}
}

func (s *Server) onUnsubscribeTopic(clientID string, topic string) {
	s.LogDebugf("client %s unsubscribed from %s", clientID, topic)

	// remove /raw suffix if present
	topic = strings.TrimSuffix(topic, "/raw")

	switch topic {

	case api.EventAPITopicCommitmentsLatest,
		api.EventAPITopicCommitmentsFinalized:
		// we don't need to unsubscribe here, because this is handled by the node bridge events anyway.

	case api.EventAPITopicBlocks,
		api.EventAPITopicBlocksValidation,
		api.EventAPITopicBlocksBasic,
		api.EventAPITopicBlocksBasicTransaction,
		api.EventAPITopicBlocksBasicTransactionTaggedData,
		api.EventAPITopicBlocksBasicTaggedData:
		s.stopListenIfNeeded(GrpcListenToBlocks)

	case api.EventAPITopicBlockMetadataAccepted:
		s.stopListenIfNeeded(GrpcListenToAcceptedBlocks)

	case api.EventAPITopicBlockMetadataConfirmed:
		s.stopListenIfNeeded(GrpcListenToConfirmedBlocks)

	default:
		switch {
		case strings.HasPrefix(topic, "blocks/basic/") && strings.Contains(topic, "tagged-data/"):
			// topicBlocksBasicTransactionTaggedDataTag
			// topicBlocksBasicTaggedDataTag
			s.stopListenIfNeeded(GrpcListenToBlocks)

		case strings.HasPrefix(topic, "block-metadata/"):
			// topicBlockMetadata
			// it can't be topicBlockMetadataAccepted or topicBlockMetadataConfirmed because they are handled above
			s.stopListenIfNeeded(GrpcListenToAcceptedBlocks)
			s.stopListenIfNeeded(GrpcListenToConfirmedBlocks)

		case strings.HasPrefix(topic, "outputs/") || strings.HasPrefix(topic, "transactions/") || strings.HasPrefix(topic, "transaction-metadata/"):
			// topicOutputs
			// topicAccountOutputs
			// topicAnchorOutputs
			// topicFoundryOutputs
			// topicNFTOutputs
			// topicOutputsByUnlockConditionAndAddress
			// topicSpentOutputsByUnlockConditionAndAddress
			// topicTransactionsIncludedBlockMetadata
			// topicTransactionMetadata
			s.stopListenIfNeeded(GrpcListenToAcceptedTransactions)
			s.stopListenIfNeeded(GrpcListenToLedgerUpdates)
		}
	}
}

func (s *Server) addGRPCSubscription(ctx context.Context, grpcCall string) *grpcSubcription {
	s.grpcSubscriptionsLock.Lock()
	defer s.grpcSubscriptionsLock.Unlock()

	if sub, ok := s.grpcSubscriptions[grpcCall]; ok {
		// subscription already exists
		// => increase count to track subscribers
		sub.count++

		return nil
	}

	ctxCancel, cancel := context.WithCancel(ctx)

	sub := &grpcSubcription{
		count:      1,
		context:    ctxCancel,
		cancelFunc: cancel,
		identifier: rand.Int(), //nolint:gosec // we do not care about weak random numbers here
	}
	s.grpcSubscriptions[grpcCall] = sub

	return sub
}

func (s *Server) removeGRPCSubscription(grpcCall string, subscriptionIdentifier int) {
	s.grpcSubscriptionsLock.Lock()
	defer s.grpcSubscriptionsLock.Unlock()

	if sub, ok := s.grpcSubscriptions[grpcCall]; ok && sub.identifier == subscriptionIdentifier {
		// Only delete if it was not already replaced by a new one.
		delete(s.grpcSubscriptions, grpcCall)
	}
}

func (s *Server) startListenIfNeeded(ctx context.Context, grpcCall string, listenFunc func(context.Context) error) {
	sub := s.addGRPCSubscription(ctx, grpcCall)
	if sub == nil {
		// subscription already exists
		return
	}

	go func() {
		s.LogInfof("Listen to %s", grpcCall)

		if err := listenFunc(sub.context); err != nil && !ierrors.Is(err, context.Canceled) {
			s.LogErrorf("Finished listen to %s with error: %s", grpcCall, err.Error())
			if status.Code(err) == codes.Unavailable && s.shutdownHandler != nil {
				s.shutdownHandler.SelfShutdown("INX became unavailable", true)
			}
		} else {
			s.LogInfof("Finished listen to %s", grpcCall)
		}

		s.removeGRPCSubscription(grpcCall, sub.identifier)
	}()
}

func (s *Server) stopListenIfNeeded(grpcCall string) {
	s.grpcSubscriptionsLock.Lock()
	defer s.grpcSubscriptionsLock.Unlock()

	sub, ok := s.grpcSubscriptions[grpcCall]
	if ok {
		// subscription found
		// decrease amount of subscribers
		sub.count--

		if sub.count == 0 {
			// => no more subscribers => stop listening
			sub.cancelFunc()
			delete(s.grpcSubscriptions, grpcCall)
		}
	}
}

func (s *Server) listenToBlocks(ctx context.Context) error {
	return s.NodeBridge.ListenToBlocks(ctx, func(block *iotago.Block, rawData []byte) error {
		if err := s.publishBlockIfSubscribed(block, rawData); err != nil {
			s.LogErrorf("failed to publish block: %v", err)
		}

		// we don't return an error here, because we want to continue listening even if publishing fails once
		return nil
	})
}

func (s *Server) listenToAcceptedBlocksMetadata(ctx context.Context) error {
	return s.NodeBridge.ListenToAcceptedBlocks(ctx, func(blockMetadata *api.BlockMetadataResponse) error {
		if err := s.publishBlockMetadataOnTopicsIfSubscribed(func() (*api.BlockMetadataResponse, error) { return blockMetadata, nil },
			api.EventAPITopicBlockMetadataAccepted,
			GetTopicBlockMetadata(blockMetadata.BlockID),
		); err != nil {
			s.LogErrorf("failed to publish accepted block metadata: %v", err)
		}

		// we don't return an error here, because we want to continue listening even if publishing fails once
		return nil
	})
}

func (s *Server) listenToConfirmedBlocksMetadata(ctx context.Context) error {
	return s.NodeBridge.ListenToConfirmedBlocks(ctx, func(blockMetadata *api.BlockMetadataResponse) error {
		if err := s.publishBlockMetadataOnTopicsIfSubscribed(func() (*api.BlockMetadataResponse, error) { return blockMetadata, nil },
			api.EventAPITopicBlockMetadataConfirmed,
			GetTopicBlockMetadata(blockMetadata.BlockID),
		); err != nil {
			s.LogErrorf("failed to publish confirmed block metadata: %v", err)
		}

		// we don't return an error here, because we want to continue listening even if publishing fails once
		return nil
	})
}

func (s *Server) listenToAcceptedTransactions(ctx context.Context) error {
	return s.NodeBridge.ListenToAcceptedTransactions(ctx, func(payload *nodebridge.AcceptedTransaction) error {
		for _, consumed := range payload.Consumed {
			if err := s.publishOutputIfSubscribed(ctx, consumed, true); err != nil {
				s.LogErrorf("failed to publish spent output in listen to accepted transaction update: %v", err)
			}
		}

		for _, created := range payload.Created {
			if err := s.publishOutputIfSubscribed(ctx, created, true); err != nil {
				s.LogErrorf("failed to publish created output in listen to accepted transaction update: %v", err)
			}
		}

		// publish the transaction metadata for this transaction in case someone subscribed to it
		s.fetchAndPublishTransactionMetadata(ctx, payload.TransactionID)

		// we don't return an error here, because we want to continue listening even if publishing fails once
		return nil
	})
}

func (s *Server) listenToLedgerUpdates(ctx context.Context) error {
	return s.NodeBridge.ListenToLedgerUpdates(ctx, 0, 0, func(payload *nodebridge.LedgerUpdate) error {
		for _, consumed := range payload.Consumed {
			if err := s.publishOutputIfSubscribed(ctx, consumed, true); err != nil {
				s.LogErrorf("failed to publish spent output in ledger update: %v", err)
			}
		}

		for _, created := range payload.Created {
			if err := s.publishOutputIfSubscribed(ctx, created, true); err != nil {
				s.LogErrorf("failed to publish created output in ledger update: %v", err)
			}

			// If this is the first output in a transaction (index 0), publish the transaction
			// metadata for the transaction that created this output in case someone subscribed to it.
			if created.OutputID.Index() == 0 {
				s.fetchAndPublishTransactionMetadata(ctx, created.OutputID.TransactionID())
			}
		}

		// we don't return an error here, because we want to continue listening even if publishing fails once
		return nil
	})
}

func (s *Server) fetchAndPublishLatestCommitmentTopic() {
	if err := s.publishCommitmentOnTopicIfSubscribed(api.EventAPITopicCommitmentsLatest,
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
	if err := s.publishCommitmentOnTopicIfSubscribed(api.EventAPITopicCommitmentsFinalized,
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
	if err := s.publishBlockMetadataOnTopicsIfSubscribed(func() (*api.BlockMetadataResponse, error) {
		resp, err := s.NodeBridge.BlockMetadata(ctx, blockID)
		if err != nil {
			return nil, ierrors.Wrapf(err, "failed to retrieve block metadata %s", blockID.ToHex())
		}

		return resp, nil
	}, GetTopicBlockMetadata(blockID)); err != nil {
		s.LogErrorf("failed to publish block metadata %s: %v", blockID.ToHex(), err)
	}
}

func (s *Server) fetchAndPublishOutput(ctx context.Context, outputID iotago.OutputID) {
	// we need to fetch the output to figure out which topics we need to publish on
	output, err := s.NodeBridge.Output(ctx, outputID)
	if err != nil {
		s.LogErrorf("failed to retrieve output %s: %v", outputID.ToHex(), err)
		return
	}

	if err := s.publishOutputIfSubscribed(ctx, output, false); err != nil {
		s.LogErrorf("failed to publish output %s: %v", outputID.ToHex(), err)
	}
}

func (s *Server) fetchAndPublishTransactionMetadata(ctx context.Context, transactionID iotago.TransactionID) {
	if err := s.publishTransactionMetadataOnTopicsIfSubscribed(func() (*api.TransactionMetadataResponse, error) {
		resp, err := s.NodeBridge.TransactionMetadata(ctx, transactionID)
		if err != nil {
			return nil, ierrors.Wrapf(err, "failed to retrieve transaction metadata %s", transactionID.ToHex())
		}

		return resp, nil
	}, GetTopicTransactionMetadata(transactionID)); err != nil {
		s.LogErrorf("failed to publish transaction metadata %s: %v", transactionID.ToHex(), err)
	}
}

func (s *Server) fetchAndPublishTransactionInclusionBlockMetadata(ctx context.Context, transactionID iotago.TransactionID) {

	var blockID iotago.BlockID
	blockIDFunc := func() (iotago.BlockID, error) {
		if blockID.Empty() {
			// get the output and then the blockID of the transaction that created the output
			outputID := iotago.OutputID{}
			copy(outputID[:], transactionID[:])

			ctxFetch, cancelFetch := context.WithTimeout(ctx, fetchTimeout)
			defer cancelFetch()

			output, err := s.NodeBridge.Output(ctxFetch, outputID)
			if err != nil {
				return iotago.EmptyBlockID, ierrors.Wrapf(err, "failed to retrieve output of transaction %s", transactionID.ToHex())
			}

			return output.Metadata.BlockID, nil
		}

		return blockID, nil
	}

	s.fetchAndPublishTransactionInclusionBlockMetadataWithBlockID(ctx, transactionID, blockIDFunc)
}

func (s *Server) fetchAndPublishTransactionInclusionBlockMetadataWithBlockID(ctx context.Context, transactionID iotago.TransactionID, blockIDFunc func() (iotago.BlockID, error)) {
	ctxFetch, cancelFetch := context.WithTimeout(ctx, fetchTimeout)
	defer cancelFetch()

	var blockMetadata *api.BlockMetadataResponse
	if err := s.publishBlockMetadataOnTopicsIfSubscribed(func() (*api.BlockMetadataResponse, error) {
		if blockMetadata != nil {
			return blockMetadata, nil
		}

		blockID, err := blockIDFunc()
		if err != nil {
			return nil, err
		}

		resp, err := s.NodeBridge.BlockMetadata(ctxFetch, blockID)
		if err != nil {
			return nil, ierrors.Wrapf(err, "failed to retrieve block metadata %s", blockID.ToHex())
		}
		blockMetadata = resp

		return blockMetadata, nil
	}, GetTopicTransactionsIncludedBlockMetadata(transactionID)); err != nil {
		s.LogErrorf("failed to publish transaction inclusion %s: %v", transactionID.ToHex(), err)
	}
}
