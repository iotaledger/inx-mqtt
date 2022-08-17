package mqtt

import (
	"errors"
	"sync"
)

type OnClientConnectFunc func(clientID string)
type OnClientDisconnectFunc func(clientID string)
type OnTopicSubscribeFunc func(clientID string, topic string)
type OnTopicUnsubscribeFunc func(clientID string, topic string)
type DropClientFunc func(clientID string, reason error)

var (
	ErrMaxTopicSubscriptionsPerClientReached = errors.New("maximum amount of topic subscriptions per client reached")
)

// SubscriptionManager keeps track of subscribed topics of clients of the
// mqtt broker by subscribing to broker events.
// This allows to get notified when a client connects or disconnects
// or a topic is subscribed or unsubscribed.
type SubscriptionManager struct {
	// subscribers keeps track of the clients and their
	// subscribed topics (and the count of subscriptions per topic).
	subscribers     *ShrinkingMap[string, *ShrinkingMap[string, int]]
	subscribersLock sync.RWMutex

	maxTopicSubscriptionsPerClient int
	cleanupThresholdCount          int
	cleanupThresholdRatio          float32

	onClientConnect    OnClientConnectFunc
	onClientDisconnect OnClientDisconnectFunc
	onTopicSubscribe   OnTopicSubscribeFunc
	onTopicUnsubscribe OnTopicUnsubscribeFunc
	dropClient         DropClientFunc
}

func NewSubscriptionManager(
	onClientConnect OnClientConnectFunc,
	onClientDisconnect OnClientDisconnectFunc,
	onTopicSubscribe OnTopicSubscribeFunc,
	onTopicUnsubscribe OnTopicUnsubscribeFunc,
	dropClient DropClientFunc,
	maxTopicSubscriptionsPerClient int,
	cleanupThresholdCount int,
	cleanupThresholdRatio float32) *SubscriptionManager {

	return &SubscriptionManager{
		subscribers: New[string, *ShrinkingMap[string, int]](
			WithShrinkingThresholdRatio(cleanupThresholdRatio),
			WithShrinkingThresholdCount(cleanupThresholdCount),
		),
		onClientConnect:                onClientConnect,
		onClientDisconnect:             onClientDisconnect,
		onTopicSubscribe:               onTopicSubscribe,
		onTopicUnsubscribe:             onTopicUnsubscribe,
		dropClient:                     dropClient,
		maxTopicSubscriptionsPerClient: maxTopicSubscriptionsPerClient,
		cleanupThresholdCount:          cleanupThresholdCount,
		cleanupThresholdRatio:          cleanupThresholdRatio,
	}
}

func (s *SubscriptionManager) Connect(clientID string) {
	s.subscribersLock.Lock()
	defer s.subscribersLock.Unlock()

	// in case the client already exists, we cleanup old subscriptions.
	s.cleanupClientWithoutLocking(clientID)

	// create a new map for the client
	s.subscribers.Set(clientID, New[string, int](
		WithShrinkingThresholdRatio(s.cleanupThresholdRatio),
		WithShrinkingThresholdCount(s.cleanupThresholdCount),
	))

	if s.onClientConnect != nil {
		s.onClientConnect(clientID)
	}
}

func (s *SubscriptionManager) Disconnect(clientID string) {
	s.subscribersLock.Lock()
	defer s.subscribersLock.Unlock()

	// cleanup the client
	s.cleanupClientWithoutLocking(clientID)

	// send disconnect notification then delete the subscriber
	if s.onClientDisconnect != nil {
		s.onClientDisconnect(clientID)
	}
}

func (s *SubscriptionManager) Subscribe(clientID string, topic string) {
	s.subscribersLock.Lock()
	defer s.subscribersLock.Unlock()

	// check if the client is connected
	subscribedTopics, has := s.subscribers.Get(clientID)
	if !has {
		return
	}

	count, has := subscribedTopics.Get(topic)
	if has {
		subscribedTopics.Set(topic, count+1)
	} else {
		// add a new topic
		subscribedTopics.Set(topic, 1)

		// check if the client has reached the max number of subscriptions
		if subscribedTopics.Size() >= s.maxTopicSubscriptionsPerClient {
			// cleanup the client
			s.cleanupClientWithoutLocking(clientID)
			// drop the client
			if s.dropClient != nil {
				s.dropClient(clientID, ErrMaxTopicSubscriptionsPerClientReached)
			}
		}
	}

	if s.onTopicSubscribe != nil {
		s.onTopicSubscribe(clientID, topic)
	}
}

func (s *SubscriptionManager) Unsubscribe(clientID string, topic string) {
	s.subscribersLock.Lock()
	defer s.subscribersLock.Unlock()

	// check if the client is connected
	subscribedTopics, has := s.subscribers.Get(clientID)
	if !has {
		return
	}

	count, has := subscribedTopics.Get(topic)
	if has {
		if count <= 1 {
			// delete the topic
			subscribedTopics.Delete(topic)
		} else {
			subscribedTopics.Set(topic, count-1)
		}
	}

	if s.onTopicUnsubscribe != nil {
		s.onTopicUnsubscribe(clientID, topic)
	}
}

func (s *SubscriptionManager) HasSubscribers(topic string) bool {
	s.subscribersLock.RLock()
	defer s.subscribersLock.RUnlock()

	hasSubscribers := false

	// loop over all clients
	s.subscribers.ForEach(func(clientID string, topics *ShrinkingMap[string, int]) bool {
		count, has := topics.Get(topic)
		if has && count > 0 {
			hasSubscribers = true

			return false
		}

		return true
	})

	return hasSubscribers
}

// SubscribersSize returns the size of the underlying map of the SubscriptionManager.
func (s *SubscriptionManager) SubscribersSize() int {
	s.subscribersLock.RLock()
	defer s.subscribersLock.RUnlock()

	return s.subscribers.Size()
}

// TopicsSize returns the size of all underlying maps of the SubscriptionManager.
func (s *SubscriptionManager) TopicsSize() int {
	s.subscribersLock.RLock()
	defer s.subscribersLock.RUnlock()

	count := 0

	// loop over all clients
	s.subscribers.ForEach(func(clientID string, topics *ShrinkingMap[string, int]) bool {
		count += topics.Size()

		return true
	})

	return count
}

// cleanupClientWithoutLocking removes all subscriptions and the client itself.
func (s *SubscriptionManager) cleanupClientWithoutLocking(clientID string) {

	// check if the client exists
	subscribedTopics, has := s.subscribers.Get(clientID)
	if !has {
		return
	}

	// loop over all topics and delete them
	subscribedTopics.ForEach(func(topic string, count int) bool {
		if s.onTopicUnsubscribe != nil {
			// call the topic unsubscribe as many times as it was subscribed
			for i := 0; i < count; i++ {
				s.onTopicUnsubscribe(clientID, topic)
			}
		}

		// delete the topic
		subscribedTopics.Delete(topic)

		return true
	})

	// delete the client
	s.subscribers.Delete(clientID)
}
