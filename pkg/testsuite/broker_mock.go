//nolint:revive // skip linter for this package name
package testsuite

import (
	"sync"
	"testing"

	"github.com/mochi-co/mqtt/server/system"
	"github.com/stretchr/testify/require"

	"github.com/iotaledger/hive.go/web/subscriptionmanager"
	"github.com/iotaledger/inx-mqtt/pkg/broker"
)

type MockedBroker struct {
	t *testing.T

	hasSubscribersCallback func(topic string)
	subscriptionmanager    *subscriptionmanager.SubscriptionManager[string, string]

	mockedSubscribedTopicsAndClientsLock sync.RWMutex
	mockedSubscribedTopicsAndClients     map[string]map[string]func(topic string, payload []byte)
}

var _ broker.Broker = &MockedBroker{}

func NewMockedBroker(t *testing.T) *MockedBroker {
	t.Helper()

	broker := &MockedBroker{
		t:                                t,
		hasSubscribersCallback:           nil,
		subscriptionmanager:              subscriptionmanager.New[string, string](),
		mockedSubscribedTopicsAndClients: make(map[string]map[string]func(topic string, payload []byte)),
	}

	return broker
}

//
// Broker interface
//

func (m *MockedBroker) Events() *subscriptionmanager.Events[string, string] {
	return m.subscriptionmanager.Events()
}

func (m *MockedBroker) Start() error {
	return nil
}

func (m *MockedBroker) Stop() error {
	return nil
}

func (m *MockedBroker) HasSubscribers(topic string) bool {
	// this callback is used in the testsuite to check if a message is
	// about to be sent to a topic that was not expected to have subscribers
	if m.hasSubscribersCallback != nil {
		m.hasSubscribersCallback(topic)
	}

	return m.subscriptionmanager.TopicHasSubscribers(topic)
}

func (m *MockedBroker) Send(topic string, payload []byte) error {
	m.mockedSubscribedTopicsAndClientsLock.RLock()
	defer m.mockedSubscribedTopicsAndClientsLock.RUnlock()

	if _, ok := m.mockedSubscribedTopicsAndClients[topic]; ok {
		// send to all subscribers
		for _, callback := range m.mockedSubscribedTopicsAndClients[topic] {
			if callback != nil {
				callback(topic, payload)
			}
		}
	}

	return nil
}

func (m *MockedBroker) SystemInfo() *system.Info {
	panic("not implemented")
}

func (m *MockedBroker) SubscribersSize() int {
	return m.subscriptionmanager.SubscribersSize()
}

func (m *MockedBroker) TopicsSize() int {
	return m.subscriptionmanager.TopicsSize()
}

//
// Mock functions
//

func (m *MockedBroker) MockClear() {
	m.hasSubscribersCallback = nil

	// we can't replace the subscriptionmanager, otherwise the events will not be wired correctly
	// so we need to manually disconnect all clients and remove all subscriptions
	clientIDs := make(map[string]struct{})
	for _, clients := range m.mockedSubscribedTopicsAndClients {
		for clientID := range clients {
			clientIDs[clientID] = struct{}{}
		}
	}

	for clientID := range clientIDs {
		m.MockClientDisconnected(clientID)
	}
	require.Equal(m.t, m.subscriptionmanager.TopicsSize(), 0, "topics not empty")
	require.Equal(m.t, m.subscriptionmanager.SubscribersSize(), 0, "subscribers not empty")

	m.mockedSubscribedTopicsAndClients = make(map[string]map[string]func(topic string, payload []byte))
}
func (m *MockedBroker) MockSetHasSubscribersCallback(hasSubscribersCallback func(topic string)) {
	m.hasSubscribersCallback = hasSubscribersCallback
}

func (m *MockedBroker) MockClientConnected(clientID string) {
	m.subscriptionmanager.Connect(clientID)
}

func (m *MockedBroker) MockClientDisconnected(clientID string) {
	m.mockedSubscribedTopicsAndClientsLock.Lock()
	defer m.mockedSubscribedTopicsAndClientsLock.Unlock()

	if !m.subscriptionmanager.Disconnect(clientID) {
		require.FailNow(m.t, "client was not connected")
		return
	}

	// client was disconnected, so we need to remove all subscriptions
	for topic, clients := range m.mockedSubscribedTopicsAndClients {
		if _, exists := clients[clientID]; exists {
			delete(clients, clientID)
			if len(clients) == 0 {
				delete(m.mockedSubscribedTopicsAndClients, topic)
			}
		}
	}
}

func (m *MockedBroker) MockTopicSubscribed(clientID string, topic string, callback func(topic string, payload []byte)) {
	m.mockedSubscribedTopicsAndClientsLock.Lock()
	defer m.mockedSubscribedTopicsAndClientsLock.Unlock()

	if !m.subscriptionmanager.Subscribe(clientID, topic) {
		require.FailNow(m.t, "subscription failed")
		return
	}

	// topic was subscribed, so we need to add the callback
	if _, ok := m.mockedSubscribedTopicsAndClients[topic]; !ok {
		m.mockedSubscribedTopicsAndClients[topic] = make(map[string]func(topic string, payload []byte))
	}
	m.mockedSubscribedTopicsAndClients[topic][clientID] = callback
}

func (m *MockedBroker) MockTopicUnsubscribed(clientID string, topic string) {
	m.mockedSubscribedTopicsAndClientsLock.Lock()
	defer m.mockedSubscribedTopicsAndClientsLock.Unlock()

	if !m.subscriptionmanager.Unsubscribe(clientID, topic) {
		require.FailNow(m.t, "unsubscription failed")
		return
	}

	// topic was unsubscribed, so we need to remove the callback
	delete(m.mockedSubscribedTopicsAndClients[topic], clientID)
	if len(m.mockedSubscribedTopicsAndClients[topic]) == 0 {
		delete(m.mockedSubscribedTopicsAndClients, topic)
	}
}
