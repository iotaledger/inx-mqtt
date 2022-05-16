package mqtt

import (
	"sync"
)

type OnSubscribeHandler func(topic string)
type OnUnsubscribeHandler func(topic string)

// topicManager keeps track of subscribed topics of the mqtt broker by subscribing to broker topic events.
// This allows to get notified when a topic is subscribed or unsubscribed
type topicManager struct {
	subscribedTopics        map[string]int
	subscribedTopicsLock    sync.RWMutex
	subscribedTopicsDeleted int

	cleanupThreshold int

	onSubscribe   OnSubscribeHandler
	onUnsubscribe OnUnsubscribeHandler
}

func (t *topicManager) Subscribe(topicName string) {
	t.subscribedTopicsLock.Lock()
	defer t.subscribedTopicsLock.Unlock()

	count, has := t.subscribedTopics[topicName]
	if has {
		t.subscribedTopics[topicName] = count + 1
	} else {
		t.subscribedTopics[topicName] = 1
	}

	if t.onSubscribe != nil {
		t.onSubscribe(topicName)
	}
}

func (t *topicManager) Unsubscribe(topicName string) {
	t.subscribedTopicsLock.Lock()
	defer t.subscribedTopicsLock.Unlock()

	count, has := t.subscribedTopics[topicName]
	if has {
		if count <= 1 {
			t.deleteTopic(topicName)
		} else {
			t.subscribedTopics[topicName] = count - 1
		}
	}

	if t.onUnsubscribe != nil {
		t.onUnsubscribe(topicName)
	}
}

// Size returns the size of the underlying map of the topics manager.
func (t *topicManager) Size() int {
	t.subscribedTopicsLock.RLock()
	defer t.subscribedTopicsLock.RUnlock()

	return len(t.subscribedTopics)
}

func (t *topicManager) hasSubscribers(topicName string) bool {
	t.subscribedTopicsLock.RLock()
	defer t.subscribedTopicsLock.RUnlock()

	count, has := t.subscribedTopics[topicName]
	return has && count > 0
}

// cleanupWithoutLocking recreates the subscribedTopics map to release memory for the garbage collector.
func (t *topicManager) cleanupWithoutLocking() {
	subscribedTopics := make(map[string]int)
	for topicName, count := range t.subscribedTopics {
		subscribedTopics[topicName] = count
	}
	t.subscribedTopics = subscribedTopics
	t.subscribedTopicsDeleted = 0
}

// deleteTopic deletes a topic from the manager.
func (t *topicManager) deleteTopic(topicName string) {
	delete(t.subscribedTopics, topicName)

	// increase the deletion counter to trigger garbage collection
	t.subscribedTopicsDeleted++
	if t.cleanupThreshold != 0 && t.subscribedTopicsDeleted >= t.cleanupThreshold {
		t.cleanupWithoutLocking()
	}
}

func newTopicManager(onSubscribe OnSubscribeHandler, onUnsubscribe OnUnsubscribeHandler, cleanupThreshold int) *topicManager {
	return &topicManager{
		subscribedTopics: make(map[string]int),
		onSubscribe:      onSubscribe,
		onUnsubscribe:    onUnsubscribe,
		cleanupThreshold: cleanupThreshold,
	}
}
