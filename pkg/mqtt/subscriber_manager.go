package mqtt

import (
	"sync"
)

type OnSubscribeFunc func(topic string, id string)
type OnUnsubscribeFunc func(topic string, id string)
type OnConnectFunc func(id string)
type OnDisconnectFunc func(id string)

// Subscriber Manager keeps track of the connected client and its own topics for server to manage subscriptions
type subscriberManager struct {
	// a map keeps client ID and topics
	subscribers    *ShrinkingMap[string, *ShrinkingMap[string, string]]
	subscriberLock sync.RWMutex

	cleanupThreshold int

	onSubscribe   OnSubscribeFunc
	onUnsubscribe OnUnsubscribeFunc
	onConnect     OnConnectFunc
	onDisconnect  OnDisconnectFunc
}

func (s *subscriberManager) Connect(clientID string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()

	// if the client ID is duplicated, kicks the old client and replease with the new one.
	// add the client ID to map
	s.subscribers.Set(clientID, New[string, string](s.cleanupThreshold))

	if s.onConnect != nil {
		s.onConnect(clientID)
	}
}

func (s *subscriberManager) Disconnect(clientID string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()
	// send disconnect notification then delete the subscriber
	if s.onDisconnect != nil {
		s.onDisconnect(clientID)
	}
	// remove the client ID from map
	s.subscribers.Delete(clientID)
}

func (s *subscriberManager) Subscribe(clientID string, topicName string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()

	// check if the client has been connected
	if topics, has := s.subscribers.Get(clientID); !has {
		t := New[string, string](s.cleanupThreshold)
		t.Set(topicName, topicName)
		s.subscribers.Set(clientID, t)
	} else {
		// add the topic to the corresponding ID
		topics.Set(topicName, topicName)
	}

	if s.onSubscribe != nil {
		s.onSubscribe(clientID, topicName)
	}
}

func (s *subscriberManager) Unsubscribe(clientID string, topicName string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()

	if topics, has := s.subscribers.Get(clientID); !has {
		return
	} else {
		topics.Delete(topicName)
	}

	if s.onUnsubscribe != nil {
		s.onUnsubscribe(clientID, topicName)
	}
}

func (s *subscriberManager) hasTopic(topicName string) bool {
	s.subscriberLock.RLock()
	defer s.subscriberLock.RUnlock()

	for _, topics := range s.subscribers.m {
		if _, has := topics.Get(topicName); has {
			return true
		}
	}
	return false
}

// Size returns the size of the underlying map of the topics manager.
func (s *subscriberManager) Size() int {

	count := 0
	for _, topics := range s.subscribers.m {
		count += topics.Size()
	}
	return count
}

// Returns topics of a subscriber
func (s *subscriberManager) Topics(clientID string) map[string]string {
	topics, _ := s.subscribers.Get(clientID)
	return topics.m
}

func (s *subscriberManager) Subscribers() int {
	return s.subscribers.Size()
}

func NewSubscriberManager(onConnect OnConnectFunc, onDisconnect OnDisconnectFunc, onSubscribe OnSubscribeFunc, onUnsubscribe OnUnsubscribeFunc, cleanupThreshold int) *subscriberManager {
	return &subscriberManager{
		subscribers:      New[string, *ShrinkingMap[string, string]](cleanupThreshold),
		onConnect:        onConnect,
		onDisconnect:     onDisconnect,
		onSubscribe:      onSubscribe,
		onUnsubscribe:    onUnsubscribe,
		cleanupThreshold: cleanupThreshold,
	}
}
