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
	subscribers    map[string]map[string]string
	subscriberLock sync.RWMutex

	cleanupThreshold int

	onSubscribe   OnSubscribeFunc
	onUnsubscribe OnUnsubscribeFunc
	onConnect     OnConnectFunc
	onDisconnect  OnDisconnectFunc
}

func (s *subscriberManager) Connect(id string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()

	// if the client ID is duplicated, kicks the old client and replease with the new one.
	// add the client ID to map
	s.subscribers[id] = make(map[string]string)
	if s.onConnect != nil {
		s.onConnect(id)
	}
}

func (s *subscriberManager) Disconnect(id string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()
	// send disconnect notification then delete the subscriber
	if s.onDisconnect != nil {
		s.onDisconnect(id)
	}
	// remove the client ID from map
	delete(s.subscribers, id)
	s.cleanupSubscriberWithoutLocking(id)
}

func (s *subscriberManager) Subscribe(id string, topicName string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()

	// check if the client has been connected
	if _, has := s.subscribers[id]; !has {
		s.subscribers[id] = make(map[string]string)
	}
	// add the topic to the corresponding ID
	s.subscribers[id][topicName] = topicName

	if s.onSubscribe != nil {
		s.onSubscribe(id, topicName)
	}
}

func (s *subscriberManager) Unsubscribe(id string, topicName string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()

	if _, has := s.subscribers[id]; !has {
		return
	}
	// remove the topic from the corresponding ID
	delete(s.subscribers[id], topicName)
	s.cleanupTopicsWithoutLocking(s.subscribers[id])

	if s.onUnsubscribe != nil {
		s.onUnsubscribe(id, topicName)
	}
}

func (s *subscriberManager) hasTopic(topicName string) bool {
	s.subscriberLock.RLock()
	defer s.subscriberLock.RUnlock()

	// check if the topic exists in all subscribers
	for _, topics := range s.subscribers {
		_, has := topics[topicName]
		if has {
			return true
		}
	}
	return false
}

// recreates the subscribers map to release memory for the garbage collector
func (s *subscriberManager) cleanupSubscriberWithoutLocking(clientID string) {
	subscribers := make(map[string]map[string]string)
	for id, topics := range s.subscribers {
		subscribers[id] = topics
	}
	s.subscribers = subscribers
}

// recreates the topics map to release memory for the garbage collector
func (s *subscriberManager) cleanupTopicsWithoutLocking(subscriber map[string]string) {
	tocpis := make(map[string]string)
	for k, v := range subscriber {
		tocpis[k] = v
	}
	subscriber = tocpis
}

// Size returns the size of the underlying map of the topics manager.
func (s *subscriberManager) Size() int {

	count := 0
	for _, topics := range s.subscribers {
		count += len(topics)
	}
	return count
}

// Returns topics of a subscriber
func (s *subscriberManager) Topics(id string) map[string]string {
	return s.subscribers[id]
}

func (s *subscriberManager) Subscribers() int {
	return len(s.subscribers)
}

func NewSubscriberManager(onConnect OnConnectFunc, onDisconnect OnDisconnectFunc, onSubscribe OnSubscribeFunc, onUnsubscribe OnUnsubscribeFunc, cleanupThreshold int) *subscriberManager {
	return &subscriberManager{
		subscribers:      make(map[string]map[string]string),
		onConnect:        onConnect,
		onDisconnect:     onDisconnect,
		onSubscribe:      onSubscribe,
		onUnsubscribe:    onUnsubscribe,
		cleanupThreshold: cleanupThreshold,
	}
}
