package mqtt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/iotaledger/inx-mqtt/pkg/mqtt"
)

const (
	clientID1 = "client1"
	clientID2 = "client2"

	topicName1 = "topic1"
	topicName2 = "topic2"
)

func TestConnectWithNoTopics(t *testing.T) {
	subscriberManager := mqtt.NewSubscriberManager(nil, nil, nil, nil, 0, 0.0)
	assert.Equal(t, subscriberManager.Subscribers(), 0)
	assert.Equal(t, subscriberManager.Size(), 0)

	subscriberManager.Connect(clientID1)
	assert.Equal(t, subscriberManager.Subscribers(), 1)
	assert.Equal(t, subscriberManager.Size(), 0)

	subscriberManager.Disconnect(clientID1)
	assert.Equal(t, subscriberManager.Subscribers(), 0)
	assert.Equal(t, subscriberManager.Size(), 0)
}

func TestConnectWithSameID(t *testing.T) {
	subscriberManager := mqtt.NewSubscriberManager(nil, nil, nil, nil, 0, 0.0)
	subscriberManager.Connect(clientID1)
	subscriberManager.Subscribe(clientID1, topicName1)
	assert.Equal(t, subscriberManager.Subscribers(), 1)
	assert.Equal(t, subscriberManager.Size(), 1)

	subscriberManager.Connect(clientID1)
	subscriberManager.Subscribe(clientID1, topicName2)
	assert.Equal(t, subscriberManager.Subscribers(), 1)
	assert.Equal(t, subscriberManager.Size(), 1)
	_, has := subscriberManager.Topics(clientID1)[topicName2]
	assert.Equal(t, has, true)

	subscriberManager.Disconnect(clientID1)
	assert.Equal(t, subscriberManager.Subscribers(), 0)
	assert.Equal(t, subscriberManager.Size(), 0)
}

func TestSubscribeWithoutConnect(t *testing.T) {
	subscriberManager := mqtt.NewSubscriberManager(nil, nil, nil, nil, 0, 0.0)
	subscriberManager.Subscribe(clientID1, topicName1)
	assert.Equal(t, subscriberManager.Subscribers(), 1)
	assert.Equal(t, subscriberManager.Size(), 1)
}

func TestSubscribeWithSameTopic(t *testing.T) {
	subscriberManager := mqtt.NewSubscriberManager(nil, nil, nil, nil, 0, 0.0)
	subscriberManager.Connect(clientID1)
	subscriberManager.Subscribe(clientID1, topicName1)
	assert.Equal(t, subscriberManager.Subscribers(), 1)
	assert.Equal(t, subscriberManager.Size(), 1)

	subscriberManager.Subscribe(clientID1, topicName1)
	assert.Equal(t, subscriberManager.Subscribers(), 1)
	assert.Equal(t, subscriberManager.Size(), 1)
}

func TestUnsubscribeWithoutConnect(t *testing.T) {
	subscriberManager := mqtt.NewSubscriberManager(nil, nil, nil, nil, 0, 0.0)
	subscriberManager.Unsubscribe(clientID1, topicName1)
	assert.Equal(t, subscriberManager.Subscribers(), 0)
	assert.Equal(t, subscriberManager.Size(), 0)
}

func TestUnsubscribeWithSameTopic(t *testing.T) {
	subscriberManager := mqtt.NewSubscriberManager(nil, nil, nil, nil, 0, 0.0)
	subscriberManager.Connect(clientID1)
	subscriberManager.Subscribe(clientID1, topicName1)
	assert.Equal(t, subscriberManager.Subscribers(), 1)
	assert.Equal(t, subscriberManager.Size(), 1)

	subscriberManager.Unsubscribe(clientID1, topicName1)
	assert.Equal(t, subscriberManager.Subscribers(), 1)
	assert.Equal(t, subscriberManager.Size(), 0)

	subscriberManager.Unsubscribe(clientID1, topicName1)
	assert.Equal(t, subscriberManager.Subscribers(), 1)
	assert.Equal(t, subscriberManager.Size(), 0)
}

func TestSubscribers(t *testing.T) {
	subscriberManager := mqtt.NewSubscriberManager(nil, nil, nil, nil, 0, 0.0)
	subscriberManager.Connect(clientID1)
	subscriberManager.Connect(clientID1)
	assert.Equal(t, subscriberManager.Subscribers(), 1)
	assert.Equal(t, subscriberManager.Size(), 0)

	subscriberManager.Connect(clientID2)
	subscriberManager.Subscribe(clientID2, topicName1)
	assert.Equal(t, subscriberManager.Subscribers(), 2)
	assert.Equal(t, subscriberManager.Size(), 1)

	subscriberManager.Subscribe(clientID2, topicName2)
	assert.Equal(t, subscriberManager.Subscribers(), 2)
	assert.Equal(t, subscriberManager.Size(), 2)

	subscriberManager.Disconnect(clientID2)
	assert.Equal(t, subscriberManager.Subscribers(), 1)
	assert.Equal(t, subscriberManager.Size(), 0)
}
