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

	"github.com/gohornet/inx-mqtt/pkg/mqtt"
	"github.com/gohornet/inx-mqtt/pkg/nodebridge"
	"github.com/iotaledger/hive.go/app/core/shutdown"
	"github.com/iotaledger/hive.go/logger"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

const (
	grpcListenToLatestMilestone    = "INX.ListenToLatestMilestone"
	grpcListenToConfirmedMilestone = "INX.ListenToConfirmedMilestone"
	grpcListenToMessages           = "INX.ListenToMessages"
	grpcListenToSolidMessages      = "INX.ListenToSolidMessages"
	grpcListenToReferencedMessages = "INX.ListenToReferencedMessages"
	grpcListenToLedgerUpdates      = "INX.ListenToLedgerUpdates"
	grpcListenToMigrationReceipts  = "INX.ListenToMigrationReceipts"
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
		func(topicName string) {
			s.onSubscribeTopic(ctx, topicName)
		}, func(topicName string) {
			s.onUnsubscribeTopic(topicName)
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

	<-ctx.Done()

	if s.brokerOptions.WebsocketEnabled {
		s.LogInfo("Removing API route...")
		if err := deps.NodeBridge.UnregisterAPIRoute(APIRoute); err != nil {
			s.LogErrorf("failed to remove API route via INX: %s", err.Error())
		}
	}

	s.MQTTBroker.Stop()
}

func (s *Server) onSubscribeTopic(ctx context.Context, topic string) {
	switch topic {
	case topicMilestoneInfoLatest:
		s.startListenIfNeeded(ctx, grpcListenToLatestMilestone, s.listenToLatestMilestone)
		go s.fetchAndPublishMilestoneTopics(ctx)

	case topicMilestoneInfoConfirmed:
		s.startListenIfNeeded(ctx, grpcListenToConfirmedMilestone, s.listenToConfirmedMilestone)
		go s.fetchAndPublishMilestoneTopics(ctx)

	case topicMessages, topicMessagesTransaction, topicMessagesTransactionTaggedData, topicMessagesTaggedData, topicMilestones:
		s.startListenIfNeeded(ctx, grpcListenToMessages, s.listenToMessages)

	case topicReceipts:
		s.startListenIfNeeded(ctx, grpcListenToMigrationReceipts, s.listenToMigrationReceipts)

	default:
		if strings.HasPrefix(topic, "message-metadata/") {
			s.startListenIfNeeded(ctx, grpcListenToSolidMessages, s.listenToSolidMessages)
			s.startListenIfNeeded(ctx, grpcListenToReferencedMessages, s.listenToReferencedMessages)

			if messageID := messageIDFromMessageMetadataTopic(topic); messageID != nil {
				go s.fetchAndPublishMessageMetadata(ctx, *messageID)
			}

		} else if strings.HasPrefix(topic, "messages/") && strings.Contains(topic, "tagged-data") {
			s.startListenIfNeeded(ctx, grpcListenToMessages, s.listenToMessages)

		} else if strings.HasPrefix(topic, "outputs/") || strings.HasPrefix(topic, "transactions/") {
			s.startListenIfNeeded(ctx, grpcListenToLedgerUpdates, s.listenToLedgerUpdates)

			if transactionID := transactionIDFromTransactionsIncludedMessageTopic(topic); transactionID != nil {
				go s.fetchAndPublishTransactionInclusion(ctx, transactionID)
			}
			if outputID := outputIDFromOutputsTopic(topic); outputID != nil {
				go s.fetchAndPublishOutput(ctx, outputID)
			}
		}
	}
}

func (s *Server) onUnsubscribeTopic(topic string) {
	switch topic {
	case topicMilestoneInfoLatest:
		s.stopListenIfNeeded(grpcListenToLatestMilestone)

	case topicMilestoneInfoConfirmed:
		s.stopListenIfNeeded(grpcListenToConfirmedMilestone)

	case topicMessages, topicMessagesTransaction, topicMessagesTransactionTaggedData, topicMessagesTaggedData, topicMilestones:
		s.stopListenIfNeeded(grpcListenToMessages)

	case topicReceipts:
		s.stopListenIfNeeded(grpcListenToMigrationReceipts)

	default:
		if strings.HasPrefix(topic, "message-metadata/") {
			s.stopListenIfNeeded(grpcListenToSolidMessages)
			s.stopListenIfNeeded(grpcListenToReferencedMessages)

		} else if strings.HasPrefix(topic, "messages/") && strings.Contains(topic, "tagged-data") {
			s.stopListenIfNeeded(grpcListenToMessages)

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

func (s *Server) listenToLatestMilestone(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := s.NodeBridge.Client().ListenToLatestMilestone(c, &inx.NoParams{})
	if err != nil {
		return err
	}
	for {
		milestone, err := stream.Recv()
		if err != nil {
			if err == io.EOF || status.Code(err) == codes.Canceled {
				break
			}
			return err
		}
		if c.Err() != nil {
			break
		}
		s.PublishMilestoneOnTopic(topicMilestoneInfoLatest, milestone.GetMilestoneInfo())
	}
	return nil
}

func (s *Server) listenToConfirmedMilestone(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := s.NodeBridge.Client().ListenToConfirmedMilestone(c, &inx.NoParams{})
	if err != nil {
		return err
	}
	for {
		milestone, err := stream.Recv()
		if err != nil {
			if err == io.EOF || status.Code(err) == codes.Canceled {
				break
			}
			return err
		}
		if c.Err() != nil {
			break
		}
		s.PublishMilestoneOnTopic(topicMilestoneInfoConfirmed, milestone.GetMilestoneInfo())
	}
	return nil
}

func (s *Server) listenToMessages(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()
	filter := &inx.MessageFilter{}
	stream, err := s.NodeBridge.Client().ListenToMessages(c, filter)
	if err != nil {
		return err
	}
	for {
		message, err := stream.Recv()
		if err != nil {
			if err == io.EOF || status.Code(err) == codes.Canceled {
				break
			}
			return err
		}
		if c.Err() != nil {
			break
		}
		s.PublishMessage(message.GetMessage())
	}
	return nil
}

func (s *Server) listenToSolidMessages(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()
	filter := &inx.MessageFilter{}
	stream, err := s.NodeBridge.Client().ListenToSolidMessages(c, filter)
	if err != nil {
		return err
	}
	for {
		messageMetadata, err := stream.Recv()
		if err != nil {
			if err == io.EOF || status.Code(err) == codes.Canceled {
				break
			}
			return err
		}
		if c.Err() != nil {
			break
		}
		s.PublishMessageMetadata(messageMetadata)
	}
	return nil
}

func (s *Server) listenToReferencedMessages(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()
	filter := &inx.MessageFilter{}
	stream, err := s.NodeBridge.Client().ListenToReferencedMessages(c, filter)
	if err != nil {
		return err
	}
	for {
		messageMetadata, err := stream.Recv()
		if err != nil {
			if err == io.EOF || status.Code(err) == codes.Canceled {
				break
			}
			return err
		}
		if c.Err() != nil {
			break
		}
		s.PublishMessageMetadata(messageMetadata)
	}
	return nil
}

func (s *Server) listenToLedgerUpdates(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()
	filter := &inx.LedgerRequest{}
	stream, err := s.NodeBridge.Client().ListenToLedgerUpdates(c, filter)
	if err != nil {
		return err
	}
	for {
		ledgerUpdate, err := stream.Recv()
		if err != nil {
			if err == io.EOF || status.Code(err) == codes.Canceled {
				break
			}
			return err
		}
		if c.Err() != nil {
			break
		}
		index := ledgerUpdate.GetMilestoneIndex()
		created := ledgerUpdate.GetCreated()
		consumed := ledgerUpdate.GetConsumed()
		for _, o := range created {
			s.PublishOutput(index, o)
		}
		for _, o := range consumed {
			s.PublishSpent(index, o)
		}
	}
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
			if err == io.EOF || status.Code(err) == codes.Canceled {
				break
			}
			return err
		}
		if c.Err() != nil {
			break
		}
		s.PublishReceipt(receipt)
	}
	return nil
}

func (s *Server) fetchAndPublishMilestoneTopics(ctx context.Context) {
	s.LogDebug("fetchAndPublishMilestoneTopics")
	resp, err := s.NodeBridge.Client().ReadNodeStatus(ctx, &inx.NoParams{})
	if err != nil {
		return
	}
	s.PublishMilestoneOnTopic(topicMilestoneInfoLatest, resp.GetLatestMilestone())
	s.PublishMilestoneOnTopic(topicMilestoneInfoConfirmed, resp.GetConfirmedMilestone())
}

func (s *Server) fetchAndPublishMessageMetadata(ctx context.Context, messageID iotago.MessageID) {
	s.LogDebugf("fetchAndPublishMessageMetadata: %s", iotago.MessageIDToHexString(messageID))
	resp, err := s.NodeBridge.Client().ReadMessageMetadata(ctx, inx.NewMessageId(messageID))
	if err != nil {
		return
	}
	s.PublishMessageMetadata(resp)
}

func (s *Server) fetchAndPublishOutput(ctx context.Context, outputID *iotago.OutputID) {
	s.LogDebugf("fetchAndPublishOutput: %s", outputID.ToHex())
	resp, err := s.NodeBridge.Client().ReadOutput(ctx, inx.NewOutputId(outputID))
	if err != nil {
		return
	}
	s.PublishOutput(resp.GetLedgerIndex(), resp.GetOutput())
}

func (s *Server) fetchAndPublishTransactionInclusion(ctx context.Context, transactionID *iotago.TransactionID) {
	s.LogDebugf("fetchAndPublishTransactionInclusion: %s", transactionID.ToHex())
	outputID := &iotago.OutputID{}
	copy(outputID[:], transactionID[:])

	resp, err := s.NodeBridge.Client().ReadOutput(ctx, inx.NewOutputId(outputID))
	if err != nil {
		return
	}
	s.fetchAndPublishTransactionInclusionWithMessage(ctx, transactionID, resp.GetOutput().UnwrapMessageID())
}

func (s *Server) fetchAndPublishTransactionInclusionWithMessage(ctx context.Context, transactionID *iotago.TransactionID, messageID iotago.MessageID) {
	resp, err := s.NodeBridge.Client().ReadMessage(ctx, inx.NewMessageId(messageID))
	if err != nil {
		return
	}
	s.PublishTransactionIncludedMessage(transactionID, resp)
}
