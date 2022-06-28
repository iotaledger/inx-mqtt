package mqtt

import (
	"fmt"
	"sync"
)

type OnSubscribeH func(topic string, id string)
type OnUnsubscribeH func(topic string, id string)
type OnConnectH func(id string)
type OnDisconnectH func(id string)

type subscriberManager struct {
	// a map keeps client ID and topics
	subscribers    map[string]map[string]string
	subscriberLock sync.RWMutex

	cleanupThreshold int

	onSubscribe   OnSubscribeH
	onUnsubscribe OnUnsubscribeH
	onConnect     OnConnectH
	onDisconnect  OnDisconnectH
}

func (s *subscriberManager) Connect(id string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()
	// add the client ID to map
	s.subscribers[id] = make(map[string]string)
	if s.onConnect != nil {
		s.onConnect(id)
	}
}

func (s *subscriberManager) Disconnect(id string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()
	// remove the client ID from map
	delete(s.subscribers, id)

	if s.onDisconnect != nil {
		s.onDisconnect(id)
	}
}

func (s *subscriberManager) Subscribe(id string, topicName string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()
	// add the topic to a crrosponding ID
	s.subscribers[id][topicName] = topicName

	if s.onSubscribe != nil {
		s.onSubscribe(id, topicName)
	}
}

func (s *subscriberManager) Unsubscribe(id string, topicName string) {
	s.subscriberLock.Lock()
	defer s.subscriberLock.Unlock()
	// remove the topic from a crrosponding ID
	delete(s.subscribers[id], topicName)

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
		if has == true {
			return true
		}
	}
	return false
}

// Size returns the size of the underlying map of the topics manager.
func (s *subscriberManager) Size() int {

	count := 0
	for _, topics := range s.subscribers {
		count += len(topics)
	}
	return count
}

func (s *subscriberManager) DumpSubscribers() {
	fmt.Println("========Dump==========")
	for id, topics := range s.subscribers {
		fmt.Println("ID: ", id)
		for _, topic := range topics {
			fmt.Println(topic)
		}
	}
	fmt.Println("======================")
}

func newSubscriberManager(onConnect OnConnectH, onDisconnect OnDisconnectH, onSubscribe OnSubscribeH, onUnsubscribe OnUnsubscribeH, cleanupThreshold int) *subscriberManager {
	return &subscriberManager{
		subscribers:      make(map[string]map[string]string),
		onConnect:        onConnect,
		onDisconnect:     onDisconnect,
		onSubscribe:      onSubscribe,
		onUnsubscribe:    onUnsubscribe,
		cleanupThreshold: cleanupThreshold,
	}
}
