package mqtt

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	mqtt "github.com/mochi-co/mqtt/server"
	"github.com/mochi-co/mqtt/server/events"
	"github.com/mochi-co/mqtt/server/listeners"
	"github.com/mochi-co/mqtt/server/listeners/auth"
	"github.com/mochi-co/mqtt/server/system"
)

// Broker is a simple mqtt publisher abstraction.
type Broker struct {
	broker              *mqtt.Server
	opts                *BrokerOptions
	subscriptionManager *SubscriptionManager
}

// NewBroker creates a new broker.
func NewBroker(onClientConnect OnClientConnectFunc, onClientDisconnect OnClientDisconnectFunc, onTopicSubscribe OnTopicSubscribeFunc, onTopicUnsubscribe OnTopicUnsubscribeFunc, brokerOpts *BrokerOptions) (*Broker, error) {

	if !brokerOpts.WebsocketEnabled && !brokerOpts.TCPEnabled {
		return nil, errors.New("at least websocket or TCP must be enabled")
	}

	broker := mqtt.NewServer(&mqtt.Options{
		BufferSize:      brokerOpts.BufferSize,
		BufferBlockSize: brokerOpts.BufferBlockSize,
	})

	if brokerOpts.WebsocketEnabled {
		// check websocket bind address
		_, _, err := net.SplitHostPort(brokerOpts.WebsocketBindAddress)
		if err != nil {
			return nil, fmt.Errorf("parsing websocket bind address (%s) failed: %w", brokerOpts.WebsocketBindAddress, err)
		}

		ws := listeners.NewWebsocket("ws1", brokerOpts.WebsocketBindAddress)
		if err := broker.AddListener(ws, &listeners.Config{
			Auth: &AuthAllowEveryone{},
			TLS:  nil,
		}); err != nil {
			return nil, fmt.Errorf("adding websocket listener failed: %w", err)
		}
	}

	if brokerOpts.TCPEnabled {
		// check tcp bind address
		_, _, err := net.SplitHostPort(brokerOpts.TCPBindAddress)
		if err != nil {
			return nil, fmt.Errorf("parsing TCP bind address (%s) failed: %w", brokerOpts.TCPBindAddress, err)
		}

		tcp := listeners.NewTCP("t1", brokerOpts.TCPBindAddress)

		var tcpAuthController auth.Controller
		if brokerOpts.TCPAuthEnabled {
			var err error
			tcpAuthController, err = NewAuthAllowUsers(brokerOpts.TCPAuthPasswordSalt, brokerOpts.TCPAuthUsers)
			if err != nil {
				return nil, fmt.Errorf("enabling TCP Authentication failed: %w", err)
			}
		} else {
			tcpAuthController = &AuthAllowEveryone{}
		}

		var tlsConfig *tls.Config
		if brokerOpts.TCPTLSEnabled {
			var err error

			tlsConfig, err = NewTLSConfig(brokerOpts.TCPTLSCertificatePath, brokerOpts.TCPTLSPrivateKeyPath)
			if err != nil {
				return nil, fmt.Errorf("enabling TCP TLS failed: %w", err)
			}
		}

		if err := broker.AddListener(tcp, &listeners.Config{
			Auth:      tcpAuthController,
			TLSConfig: tlsConfig,
		}); err != nil {
			return nil, fmt.Errorf("adding TCP listener failed: %w", err)
		}
	}

	// this function is used to drop malicious clients
	dropClient := func(clientID string, reason error) {
		client, exists := broker.Clients.Get(clientID)
		if !exists {
			return
		}

		// stop the client connection
		client.Stop(reason)

		// delete the client from the broker
		broker.Clients.Delete(clientID)
	}

	s := NewSubscriptionManager(
		onClientConnect,
		onClientDisconnect,
		onTopicSubscribe,
		onTopicUnsubscribe,
		dropClient,
		brokerOpts.MaxTopicSubscriptionsPerClient,
		brokerOpts.TopicCleanupThresholdCount,
		brokerOpts.TopicCleanupThresholdRatio,
	)

	// bind the broker events to the SubscriptionManager to track the subscriptions
	broker.Events.OnConnect = func(cl events.Client, pk events.Packet) {
		s.Connect(cl.ID)
	}

	broker.Events.OnDisconnect = func(cl events.Client, err error) {
		s.Disconnect(cl.ID)
	}

	broker.Events.OnSubscribe = func(topic string, cl events.Client, qos byte) {
		s.Subscribe(cl.ID, topic)
	}

	broker.Events.OnUnsubscribe = func(topic string, cl events.Client) {
		s.Unsubscribe(cl.ID, topic)
	}

	return &Broker{
		broker:              broker,
		opts:                brokerOpts,
		subscriptionManager: s,
	}, nil
}

// Start the broker.
func (b *Broker) Start() error {
	return b.broker.Serve()
}

// Stop the broker.
func (b *Broker) Stop() error {
	return b.broker.Close()
}

// SystemInfo returns the metrics of the broker.
func (b *Broker) SystemInfo() *system.Info {
	return b.broker.System
}

func (b *Broker) HasSubscribers(topic string) bool {
	return b.subscriptionManager.HasSubscribers(topic)
}

// Send publishes a message.
func (b *Broker) Send(topic string, payload []byte) error {
	return b.broker.Publish(topic, payload, false)
}

// SubscribersSize returns the size of the underlying map of the SubscriptionManager.
func (b *Broker) SubscribersSize() int {
	return b.subscriptionManager.SubscribersSize()
}

// TopicsSize returns the size of all underlying maps of the SubscriptionManager.
func (b *Broker) TopicsSize() int {
	return b.subscriptionManager.TopicsSize()
}
