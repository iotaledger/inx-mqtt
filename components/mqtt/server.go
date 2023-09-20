package mqtt

import (
	"context"
	"io"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/iotaledger/hive.go/app/shutdown"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/web/subscriptionmanager"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	"github.com/iotaledger/inx-mqtt/pkg/mqtt"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v4"
)

const (
	grpcListenToBlocks          = "INX.ListenToBlocks"
	grpcListenToAcceptedBlocks  = "INX.ListenToAcceptedBlocks"
	grpcListenToConfirmedBlocks = "INX.ListenToConfirmedBlocks"
	grpcListenToLedgerUpdates   = "INX.ListenToLedgerUpdates"
)

const (
	fetchTimeout = 5 * time.Second
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
		s.NodeBridge.Events.LatestCommittedSlotChanged.Hook(func(c *nodebridge.Commitment) {
			s.PublishCommitmentInfoOnTopic(topicCommitmentInfoLatest, c.CommitmentID)
			s.PublishRawCommitmentOnTopic(topicCommitments, c.Commitment)
		}).Unhook,
		s.NodeBridge.Events.LatestFinalizedSlotChanged.Hook(func(cID iotago.CommitmentID) {
			s.PublishCommitmentInfoOnTopic(topicCommitmentInfoFinalized, cID)
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
	s.LogDebugf("%s subscribed to %s", clientID, topic)
	switch topic {

	case topicCommitmentInfoLatest:
		go s.publishLatestCommitmentInfoTopic()
	case topicCommitmentInfoFinalized:
		go s.publishFinalizedCommitmentInfoTopic()
	case topicCommitments:
		go s.publishLatestCommitmentTopic()

	case topicBlocks, topicBlocksTransaction, topicBlocksTransactionTaggedData, topicBlocksTaggedData:
		s.startListenIfNeeded(ctx, grpcListenToBlocks, s.listenToBlocks)

	default:
		switch {
		case strings.HasPrefix(topic, "block-metadata/"):
			s.startListenIfNeeded(ctx, grpcListenToAcceptedBlocks, s.listenToAcceptedBlocks)
			s.startListenIfNeeded(ctx, grpcListenToConfirmedBlocks, s.listenToConfirmedBlocks)

			if blockID := blockIDFromBlockMetadataTopic(topic); !blockID.Empty() {
				go s.fetchAndPublishBlockMetadata(ctx, blockID)
			}

		case strings.HasPrefix(topic, "blocks/") && strings.Contains(topic, "tagged-data"):
			s.startListenIfNeeded(ctx, grpcListenToBlocks, s.listenToBlocks)

		case strings.HasPrefix(topic, "outputs/") || strings.HasPrefix(topic, "transactions/"):
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
	case topicBlocks, topicBlocksTransaction, topicBlocksTransactionTaggedData, topicBlocksTaggedData:
		s.stopListenIfNeeded(grpcListenToBlocks)

	default:
		switch {
		case strings.HasPrefix(topic, "block-metadata/"):
			s.stopListenIfNeeded(grpcListenToAcceptedBlocks)
			s.stopListenIfNeeded(grpcListenToConfirmedBlocks)

		case strings.HasPrefix(topic, "blocks/") && strings.Contains(topic, "tagged-data"):
			s.stopListenIfNeeded(grpcListenToBlocks)

		case strings.HasPrefix(topic, "outputs/") || strings.HasPrefix(topic, "transactions/"):
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

	for {
		block, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				break
			}

			return err
		}
		if ctx.Err() != nil {
			break
		}
		s.PublishBlock(block.GetBlock())
	}

	//nolint:nilerr // false positive
	return nil
}

func (s *Server) listenToAcceptedBlocks(ctx context.Context) error {

	stream, err := s.NodeBridge.Client().ListenToAcceptedBlocks(ctx, &inx.NoParams{})
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
		if ctx.Err() != nil {
			break
		}
		s.PublishBlockMetadata(blockMetadata)
	}

	//nolint:nilerr // false positive
	return nil
}

func (s *Server) listenToConfirmedBlocks(ctx context.Context) error {

	stream, err := s.NodeBridge.Client().ListenToConfirmedBlocks(ctx, &inx.NoParams{})
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
		if ctx.Err() != nil {
			break
		}
		s.PublishBlockMetadata(blockMetadata)
	}

	//nolint:nilerr // false positive
	return nil
}

func (s *Server) listenToLedgerUpdates(ctx context.Context) error {
	stream, err := s.NodeBridge.Client().ListenToLedgerUpdates(ctx, &inx.SlotRangeRequest{})
	if err != nil {
		return err
	}

	var latestIndex iotago.SlotIndex
	for {
		payload, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				break
			}

			return err
		}
		if ctx.Err() != nil {
			break
		}

		switch op := payload.GetOp().(type) {
		//nolint:nosnakecase // grpc uses underscores
		case *inx.LedgerUpdate_BatchMarker:
			if op.BatchMarker.GetMarkerType() == inx.LedgerUpdate_Marker_BEGIN {
				latestIndex = iotago.SlotIndex(op.BatchMarker.Slot)
			}

		//nolint:nosnakecase // grpc uses underscores
		case *inx.LedgerUpdate_Consumed:
			s.PublishSpent(latestIndex, op.Consumed)

		//nolint:nosnakecase // grpc uses underscores
		case *inx.LedgerUpdate_Created:
			s.PublishOutput(ctx, latestIndex, op.Created, true)
		}
	}

	//nolint:nilerr // false positive
	return nil
}

func (s *Server) publishLatestCommitmentTopic() {
	s.LogDebug("publishLatestCommitmentTopic")
	latest, err := s.NodeBridge.LatestCommitment()
	if err != nil {
		s.LogErrorf("failed to retrieve latest commitment: %v", err)

		return
	}

	s.PublishRawCommitmentOnTopic(topicCommitmentInfoLatest, latest)
}

func (s *Server) publishLatestCommitmentInfoTopic() {
	s.LogDebug("publishLatestCommitmentInfoTopic")
	latest, err := s.NodeBridge.LatestCommitment()
	if err != nil {
		s.LogErrorf("failed to retrieve latest commitment: %v", err)

		return
	}

	id, err := latest.ID()
	if err != nil {
		s.LogErrorf("failed to retrieve latest commitment: %v", err)

		return
	}

	s.PublishCommitmentInfoOnTopic(topicCommitmentInfoLatest, id)

	s.LogDebug("publishLatestCommitmentTopic")
	s.PublishRawCommitmentOnTopic(topicCommitmentInfoLatest, latest)
}

func (s *Server) publishFinalizedCommitmentInfoTopic() {
	s.LogDebug("publishFinalizedCommitmentInfoTopic")
	finalized := s.NodeBridge.LatestFinalizedCommitmentID()
	s.PublishCommitmentInfoOnTopic(topicCommitmentInfoFinalized, finalized)
}

func (s *Server) fetchAndPublishBlockMetadata(ctx context.Context, blockID iotago.BlockID) {
	s.LogDebugf("fetchAndPublishBlockMetadata: %s", blockID.ToHex())
	resp, err := s.NodeBridge.Client().ReadBlockMetadata(ctx, inx.NewBlockId(blockID))
	if err != nil {
		s.LogErrorf("failed to retrieve block metadata %s: %v", blockID.ToHex(), err)

		return
	}
	s.PublishBlockMetadata(resp)
}

func (s *Server) fetchAndPublishOutput(ctx context.Context, outputID iotago.OutputID) {
	s.LogDebugf("fetchAndPublishOutput: %s", outputID.ToHex())
	resp, err := s.NodeBridge.Client().ReadOutput(ctx, inx.NewOutputId(outputID))
	if err != nil {
		s.LogErrorf("failed to retrieve output %s: %v", outputID.ToHex(), err)

		return
	}
	s.PublishOutput(ctx, resp.GetLatestCommitmentId().Unwrap().Index(), resp.GetOutput(), false)
}

func (s *Server) fetchAndPublishTransactionInclusion(ctx context.Context, transactionID iotago.TransactionID) {
	s.LogDebugf("fetchAndPublishTransactionInclusion: %s", transactionID.ToHex())
	outputID := iotago.OutputID{}
	copy(outputID[:], transactionID[:])

	resp, err := s.NodeBridge.Client().ReadOutput(ctx, inx.NewOutputId(outputID))
	if err != nil {
		s.LogErrorf("failed to retrieve output of transaction %s :%v", transactionID.ToHex(), err)

		return
	}

	ctxFetch, cancelFetch := context.WithTimeout(ctx, fetchTimeout)
	defer cancelFetch()

	s.fetchAndPublishTransactionInclusionWithBlock(ctxFetch, transactionID, resp.GetOutput().UnwrapBlockID())
}

func (s *Server) fetchAndPublishTransactionInclusionWithBlock(ctx context.Context, transactionID iotago.TransactionID, blockID iotago.BlockID) {
	s.LogDebugf("fetchAndPublishTransactionInclusionWithBlock: %s", transactionID.ToHex())
	resp, err := s.NodeBridge.Client().ReadBlock(ctx, inx.NewBlockId(blockID))
	if err != nil {
		s.LogErrorf("failed to retrieve block %s :%v", blockID.ToHex(), err)

		return
	}
	s.PublishTransactionIncludedBlock(transactionID, resp)
}
