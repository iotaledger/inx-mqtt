//nolint:golint,revive,stylecheck,nosnakecase // we don't care about these linters in test cases
package mqtt_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/inx-mqtt/pkg/mqtt"
)

const (
	clientID_1 = "client1"
	clientID_2 = "client2"

	topic_1 = "topic1"
	topic_2 = "topic2"
	topic_3 = "topic3"
	topic_4 = "topic4"
	topic_5 = "topic5"
	topic_6 = "topic6"
)

func TestSubscriptionManager_ConnectWithNoTopics(t *testing.T) {
	manager := mqtt.NewSubscriptionManager(nil, nil, nil, nil, nil, 1000, 0, 0.0)

	require.Equal(t, manager.SubscribersSize(), 0)
	require.Equal(t, manager.TopicsSize(), 0)

	manager.Connect(clientID_1)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 0)

	manager.Disconnect(clientID_1)
	require.Equal(t, manager.SubscribersSize(), 0)
	require.Equal(t, manager.TopicsSize(), 0)
}

func TestSubscriptionManager_ConnectWithSameID(t *testing.T) {
	manager := mqtt.NewSubscriptionManager(nil, nil, nil, nil, nil, 1000, 0, 0.0)

	manager.Connect(clientID_1)
	manager.Subscribe(clientID_1, topic_1)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 1)

	manager.Connect(clientID_1)
	manager.Subscribe(clientID_1, topic_2)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 1)

	manager.Disconnect(clientID_1)
	require.Equal(t, manager.SubscribersSize(), 0)
	require.Equal(t, manager.TopicsSize(), 0)
}

func TestSubscriptionManager_SubscribeWithoutConnect(t *testing.T) {
	manager := mqtt.NewSubscriptionManager(nil, nil, nil, nil, nil, 1000, 0, 0.0)

	manager.Subscribe(clientID_1, topic_1)
	require.Equal(t, manager.SubscribersSize(), 0)
	require.Equal(t, manager.TopicsSize(), 0)
}

func TestSubscriptionManager_SubscribeWithSameTopic(t *testing.T) {
	manager := mqtt.NewSubscriptionManager(nil, nil, nil, nil, nil, 1000, 0, 0.0)

	manager.Connect(clientID_1)
	manager.Subscribe(clientID_1, topic_1)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 1)

	manager.Subscribe(clientID_1, topic_1)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 1)
}

func TestSubscriptionManager_UnsubscribeWithoutConnect(t *testing.T) {
	manager := mqtt.NewSubscriptionManager(nil, nil, nil, nil, nil, 1000, 0, 0.0)

	manager.Unsubscribe(clientID_1, topic_1)
	require.Equal(t, manager.SubscribersSize(), 0)
	require.Equal(t, manager.TopicsSize(), 0)
}

func TestSubscriptionManager_UnsubscribeWithSameTopic(t *testing.T) {
	manager := mqtt.NewSubscriptionManager(nil, nil, nil, nil, nil, 1000, 0, 0.0)

	manager.Connect(clientID_1)
	manager.Subscribe(clientID_1, topic_1)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 1)

	manager.Unsubscribe(clientID_1, topic_1)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 0)

	manager.Unsubscribe(clientID_1, topic_1)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 0)
}

func TestSubscriptionManager_Subscribers(t *testing.T) {
	manager := mqtt.NewSubscriptionManager(nil, nil, nil, nil, nil, 1000, 0, 0.0)

	manager.Connect(clientID_1)
	manager.Connect(clientID_1)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 0)

	manager.Connect(clientID_2)
	manager.Subscribe(clientID_2, topic_1)
	require.Equal(t, manager.SubscribersSize(), 2)
	require.Equal(t, manager.TopicsSize(), 1)

	manager.Subscribe(clientID_2, topic_2)
	require.Equal(t, manager.SubscribersSize(), 2)
	require.Equal(t, manager.TopicsSize(), 2)

	manager.Disconnect(clientID_2)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 0)
}

func TestSubscriptionManager_ClientCleanup(t *testing.T) {

	subscribe_client_1 := 0
	unsubscribe_client_1 := 0

	onTopicsSubscribe := func(clientID, topic string) {
		if clientID == clientID_1 {
			subscribe_client_1++
		}
	}

	onTopicsUnsubscribe := func(clientID, topic string) {
		if clientID == clientID_1 {
			unsubscribe_client_1++
		}
	}

	manager := mqtt.NewSubscriptionManager(nil, nil, onTopicsSubscribe, onTopicsUnsubscribe, nil, 1000, 0, 0.0)

	manager.Connect(clientID_1)
	manager.Subscribe(clientID_1, topic_1)
	manager.Subscribe(clientID_1, topic_2)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 2)
	require.Equal(t, subscribe_client_1, 2)
	require.Equal(t, unsubscribe_client_1, 0)

	manager.Connect(clientID_1)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 0)
	require.Equal(t, subscribe_client_1, 2)
	require.Equal(t, unsubscribe_client_1, 2)

	manager.Subscribe(clientID_1, topic_1)
	manager.Subscribe(clientID_1, topic_2)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 2)
	require.Equal(t, subscribe_client_1, 4)
	require.Equal(t, unsubscribe_client_1, 2)

	manager.Disconnect(clientID_1)
	require.Equal(t, manager.SubscribersSize(), 0)
	require.Equal(t, manager.TopicsSize(), 0)
	require.Equal(t, subscribe_client_1, 4)
	require.Equal(t, unsubscribe_client_1, 4)
}

func TestSubscriptionManager_MaxTopicSubscriptionsPerClient(t *testing.T) {

	clientDropped := false
	dropClient := func(clientID string, reason error) {
		clientDropped = true
	}

	manager := mqtt.NewSubscriptionManager(nil, nil, nil, nil, dropClient, 5, 0, 0.0)

	require.Equal(t, manager.SubscribersSize(), 0)
	require.Equal(t, manager.TopicsSize(), 0)
	require.Equal(t, clientDropped, false)

	manager.Connect(clientID_1)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 0)
	require.Equal(t, clientDropped, false)

	manager.Subscribe(clientID_1, topic_1)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 1)
	require.Equal(t, clientDropped, false)

	manager.Subscribe(clientID_1, topic_2)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 2)
	require.Equal(t, clientDropped, false)

	manager.Subscribe(clientID_1, topic_3)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 3)
	require.Equal(t, clientDropped, false)

	manager.Subscribe(clientID_1, topic_4)
	require.Equal(t, manager.SubscribersSize(), 1)
	require.Equal(t, manager.TopicsSize(), 4)
	require.Equal(t, clientDropped, false)

	manager.Subscribe(clientID_1, topic_5)
	require.Equal(t, manager.SubscribersSize(), 0)
	require.Equal(t, manager.TopicsSize(), 0)
	require.Equal(t, clientDropped, true)

	manager.Subscribe(clientID_1, topic_6)
	require.Equal(t, manager.SubscribersSize(), 0)
	require.Equal(t, manager.TopicsSize(), 0)
	require.Equal(t, clientDropped, true)
}
