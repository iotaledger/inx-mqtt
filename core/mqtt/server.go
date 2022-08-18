package mqtt

import (
	"context"
	"io"
	"math/rand"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/iotaledger/hive.go/core/app/core/shutdown"
	"github.com/iotaledger/hive.go/core/events"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/inx-app/nodebridge"
	"github.com/iotaledger/inx-mqtt/pkg/mqtt"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

const (
	grpcListenToBlocks            = "INX.ListenToBlocks"
	grpcListenToSolidBlocks       = "INX.ListenToSolidBlocks"
	grpcListenToReferencedBlocks  = "INX.ListenToReferencedBlocks"
	grpcListenToLedgerUpdates     = "INX.ListenToLedgerUpdates"
	grpcListenToMigrationReceipts = "INX.ListenToMigrationReceipts"
	grpcListenToTipScoreUpdates   = "INX.ListenToTipScoreUpdates"
)

var (
	emptyOutputID      = iotago.OutputID{}
	emptyTransactionID = iotago.TransactionID{}
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
	broker, err := mqtt.NewBroker(
		func(clientID string) {
			s.onClientConnect(clientID)
		}, func(clientID string) {
			s.onClientDisconnect(clientID)
		}, func(clientID string, topic string) {
			s.onSubscribeTopic(ctx, clientID, topic)
		}, func(clientID string, topic string) {
			s.onUnsubscribeTopic(clientID, topic)
		},
		s.brokerOptions)
	if err != nil {
		panic(err)
	}

	s.MQTTBroker = broker

	go func() {
		if err := broker.Start(); err != nil {
			panic(err)
		}
	}()

	if s.brokerOptions.WebsocketEnabled {
		s.LogInfo("Registering API route...")
		if err := deps.NodeBridge.RegisterAPIRoute(APIRoute, s.brokerOptions.WebsocketBindAddress); err != nil {
			s.LogErrorf("failed to register API route via INX: %s", err.Error())
		}
	}

	onLatestMilestone := events.NewClosure(func(ms *nodebridge.Milestone) {
		s.PublishMilestoneOnTopic(topicMilestoneInfoLatest, ms)
	})

	onConfirmedMilestone := events.NewClosure(func(ms *nodebridge.Milestone) {
		s.PublishMilestoneOnTopic(topicMilestoneInfoConfirmed, ms)
	})

	s.NodeBridge.Events.LatestMilestoneChanged.Hook(onLatestMilestone)
	s.NodeBridge.Events.ConfirmedMilestoneChanged.Hook(onConfirmedMilestone)

	<-ctx.Done()

	s.NodeBridge.Events.LatestMilestoneChanged.Detach(onLatestMilestone)
	s.NodeBridge.Events.ConfirmedMilestoneChanged.Detach(onConfirmedMilestone)

	if s.brokerOptions.WebsocketEnabled {
		s.LogInfo("Removing API route...")
		if err := deps.NodeBridge.UnregisterAPIRoute(APIRoute); err != nil {
			s.LogErrorf("failed to remove API route via INX: %s", err.Error())
		}
	}

	if err := s.MQTTBroker.Stop(); err != nil {
		s.LogErrorf("failed to stop MQTT broker: %s", err.Error())
	}
}

func (s *Server) onClientConnect(clientID string) {
	s.LogDebugf("onClientConnect %s", clientID)
}

func (s *Server) onClientDisconnect(clientID string) {
	s.LogDebugf("onClientDisconnect %s", clientID)
}

func (s *Server) onSubscribeTopic(ctx context.Context, clientID string, topic string) {
	s.LogDebugf("%s subscribed to %s", clientID, topic)
	switch topic {
	case topicMilestoneInfoLatest:
		go s.publishLatestMilestoneTopic()

	case topicMilestoneInfoConfirmed:
		go s.publishConfirmedMilestoneTopic()

	case topicBlocks, topicBlocksTransaction, topicBlocksTransactionTaggedData, topicBlocksTaggedData, topicMilestones:
		s.startListenIfNeeded(ctx, grpcListenToBlocks, s.listenToBlocks)

	case topicTipScoreUpdates:
		s.startListenIfNeeded(ctx, grpcListenToTipScoreUpdates, s.listenToTipScoreUpdates)

	case topicReceipts:
		s.startListenIfNeeded(ctx, grpcListenToMigrationReceipts, s.listenToMigrationReceipts)

	default:
		if strings.HasPrefix(topic, "block-metadata/") {
			s.startListenIfNeeded(ctx, grpcListenToSolidBlocks, s.listenToSolidBlocks)
			s.startListenIfNeeded(ctx, grpcListenToReferencedBlocks, s.listenToReferencedBlocks)

			if blockID := blockIDFromBlockMetadataTopic(topic); !blockID.Empty() {
				go s.fetchAndPublishBlockMetadata(ctx, blockID)
			}

		} else if strings.HasPrefix(topic, "blocks/") && strings.Contains(topic, "tagged-data") {
			s.startListenIfNeeded(ctx, grpcListenToBlocks, s.listenToBlocks)

		} else if strings.HasPrefix(topic, "outputs/") || strings.HasPrefix(topic, "transactions/") {
			s.startListenIfNeeded(ctx, grpcListenToLedgerUpdates, s.listenToLedgerUpdates)

			if transactionID := transactionIDFromTransactionsIncludedBlockTopic(topic); transactionID != emptyTransactionID {
				go s.fetchAndPublishTransactionInclusion(ctx, transactionID)
			}
			if outputID := outputIDFromOutputsTopic(topic); outputID != emptyOutputID {
				go s.fetchAndPublishOutput(ctx, outputID)
			}
		}
	}
}

func (s *Server) onUnsubscribeTopic(clientID string, topic string) {
	s.LogDebugf("%s unsubscribed from %s", clientID, topic)
	switch topic {
	case topicBlocks, topicBlocksTransaction, topicBlocksTransactionTaggedData, topicBlocksTaggedData, topicMilestones:
		s.stopListenIfNeeded(grpcListenToBlocks)

	case topicTipScoreUpdates:
		s.stopListenIfNeeded(grpcListenToTipScoreUpdates)

	case topicReceipts:
		s.stopListenIfNeeded(grpcListenToMigrationReceipts)

	default:
		if strings.HasPrefix(topic, "block-metadata/") {
			s.stopListenIfNeeded(grpcListenToSolidBlocks)
			s.stopListenIfNeeded(grpcListenToReferencedBlocks)

		} else if strings.HasPrefix(topic, "blocks/") && strings.Contains(topic, "tagged-data") {
			s.stopListenIfNeeded(grpcListenToBlocks)

		} else if strings.HasPrefix(topic, "outputs/") || strings.HasPrefix(topic, "transactions/") {
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

	c, cancel := context.WithCancel(ctx)

	//nolint:gosec // we do not care about weak random numbers here
	subscriptionIdentifier := rand.Int()
	s.grpcSubscriptions[grpcCall] = &topicSubcription{
		Count:      1,
		CancelFunc: cancel,
		Identifier: subscriptionIdentifier,
	}
	go func() {
		s.LogInfof("Listen to %s", grpcCall)
		err := listenFunc(c)
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
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := s.NodeBridge.Client().ListenToBlocks(c, &inx.NoParams{})
	if err != nil {
		return err
	}

	for {
		block, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				break
			}

			return err
		}
		if c.Err() != nil {
			break
		}
		s.PublishBlock(block.GetBlock())
	}

	//nolint:nilerr // false positive
	return nil
}

func (s *Server) listenToSolidBlocks(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := s.NodeBridge.Client().ListenToSolidBlocks(c, &inx.NoParams{})
	if err != nil {
		return err
	}

	for {
		blockMetadata, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				break
			}

			return err
		}
		if c.Err() != nil {
			break
		}
		s.PublishBlockMetadata(blockMetadata)
	}

	//nolint:nilerr // false positive
	return nil
}

func (s *Server) listenToReferencedBlocks(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := s.NodeBridge.Client().ListenToReferencedBlocks(c, &inx.NoParams{})
	if err != nil {
		return err
	}

	for {
		blockMetadata, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				break
			}

			return err
		}
		if c.Err() != nil {
			break
		}
		s.PublishBlockMetadata(blockMetadata)
	}

	//nolint:nilerr // false positive
	return nil
}

func (s *Server) listenToTipScoreUpdates(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := s.NodeBridge.Client().ListenToTipScoreUpdates(c, &inx.NoParams{})
	if err != nil {
		return err
	}

	for {
		blockMetadata, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				break
			}

			return err
		}
		if c.Err() != nil {
			break
		}
		s.PublishBlockMetadata(blockMetadata)
	}

	//nolint:nilerr // false positive
	return nil
}

func (s *Server) listenToLedgerUpdates(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := s.NodeBridge.Client().ListenToLedgerUpdates(c, &inx.MilestoneRangeRequest{})
	if err != nil {
		return err
	}

	var latestIndex iotago.MilestoneIndex
	for {
		payload, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				break
			}

			return err
		}
		if c.Err() != nil {
			break
		}
		switch op := payload.GetOp().(type) {
		case *inx.LedgerUpdate_BatchMarker:
			if op.BatchMarker.GetMarkerType() == inx.LedgerUpdate_Marker_BEGIN {
				latestIndex = op.BatchMarker.GetMilestoneIndex()
			}
		case *inx.LedgerUpdate_Consumed:
			s.PublishSpent(latestIndex, op.Consumed)
		case *inx.LedgerUpdate_Created:
			s.PublishOutput(latestIndex, op.Created)
		}
	}

	//nolint:nilerr // false positive
	return nil
}

func (s *Server) listenToMigrationReceipts(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := s.NodeBridge.Client().ListenToMigrationReceipts(c, &inx.NoParams{})
	if err != nil {
		return err
	}

	for {
		receipt, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				break
			}

			return err
		}
		if c.Err() != nil {
			break
		}
		s.PublishReceipt(receipt)
	}

	//nolint:nilerr // false positive
	return nil
}

func (s *Server) publishLatestMilestoneTopic() {
	s.LogDebug("publishLatestMilestoneTopic")
	latest, err := s.NodeBridge.LatestMilestone()
	if err == nil {
		s.PublishMilestoneOnTopic(topicMilestoneInfoLatest, latest)
	}
}

func (s *Server) publishConfirmedMilestoneTopic() {
	s.LogDebug("publishConfirmedMilestoneTopic")
	confirmed, err := s.NodeBridge.ConfirmedMilestone()
	if err == nil {
		s.PublishMilestoneOnTopic(topicMilestoneInfoConfirmed, confirmed)
	}
}

func (s *Server) fetchAndPublishBlockMetadata(ctx context.Context, blockID iotago.BlockID) {
	s.LogDebugf("fetchAndPublishBlockMetadata: %s", blockID.ToHex())
	resp, err := s.NodeBridge.Client().ReadBlockMetadata(ctx, inx.NewBlockId(blockID))
	if err != nil {
		return
	}
	s.PublishBlockMetadata(resp)
}

func (s *Server) fetchAndPublishOutput(ctx context.Context, outputID iotago.OutputID) {
	s.LogDebugf("fetchAndPublishOutput: %s", outputID.ToHex())
	resp, err := s.NodeBridge.Client().ReadOutput(ctx, inx.NewOutputId(outputID))
	if err != nil {
		return
	}
	s.PublishOutput(resp.GetLedgerIndex(), resp.GetOutput())
}

func (s *Server) fetchAndPublishTransactionInclusion(ctx context.Context, transactionID iotago.TransactionID) {
	s.LogDebugf("fetchAndPublishTransactionInclusion: %s", transactionID.ToHex())
	outputID := iotago.OutputID{}
	copy(outputID[:], transactionID[:])

	resp, err := s.NodeBridge.Client().ReadOutput(ctx, inx.NewOutputId(outputID))
	if err != nil {
		return
	}
	s.fetchAndPublishTransactionInclusionWithBlock(ctx, transactionID, resp.GetOutput().UnwrapBlockID())
}

func (s *Server) fetchAndPublishTransactionInclusionWithBlock(ctx context.Context, transactionID iotago.TransactionID, blockID iotago.BlockID) {
	s.LogDebugf("fetchAndPublishTransactionInclusionWithBlock: %s", transactionID.ToHex())
	resp, err := s.NodeBridge.Client().ReadBlock(ctx, inx.NewBlockId(blockID))
	if err != nil {
		return
	}
	s.PublishTransactionIncludedBlock(transactionID, resp)
}
