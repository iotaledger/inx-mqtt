package broker

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

	"github.com/iotaledger/hive.go/web/subscriptionmanager"
)

type Broker interface {
	// Events returns the events of the broker.
	Events() subscriptionmanager.Events[string, string]
	// Start the broker.
	Start() error
	// Stop the broker.
	Stop() error
	// HasSubscribers returns true if the topic has subscribers.
	HasSubscribers(topic string) bool
	// Send publishes a message.
	Send(topic string, payload []byte) error
	// SystemInfo returns the metrics of the broker.
	SystemInfo() *system.Info
	// SubscribersSize returns the size of the underlying map of the SubscriptionManager.
	SubscribersSize() int
	// TopicsSize returns the size of all underlying maps of the SubscriptionManager.
	TopicsSize() int
}

// Broker is a simple mqtt publisher abstraction.
type broker struct {
	server              *mqtt.Server
	opts                *Options
	subscriptionManager *subscriptionmanager.SubscriptionManager[string, string]
	unhook              func()
}

// NewBroker creates a new broker.
func NewBroker(brokerOpts ...Option) (Broker, error) {
	opts := &Options{}
	opts.ApplyOnDefault(brokerOpts...)

	if !opts.WebsocketEnabled && !opts.TCPEnabled {
		return nil, errors.New("at least websocket or TCP must be enabled")
	}

	server := mqtt.NewServer(&mqtt.Options{
		BufferSize:      opts.BufferSize,
		BufferBlockSize: opts.BufferBlockSize,
		InflightTTL:     30,
	})

	if opts.WebsocketEnabled {
		// check websocket bind address
		_, _, err := net.SplitHostPort(opts.WebsocketBindAddress)
		if err != nil {
			return nil, fmt.Errorf("parsing websocket bind address (%s) failed: %w", opts.WebsocketBindAddress, err)
		}

		ws := listeners.NewWebsocket("ws1", opts.WebsocketBindAddress)
		if err := server.AddListener(ws, &listeners.Config{
			Auth: &AuthAllowEveryone{},
			TLS:  nil,
		}); err != nil {
			return nil, fmt.Errorf("adding websocket listener failed: %w", err)
		}
	}

	if opts.TCPEnabled {
		// check tcp bind address
		_, _, err := net.SplitHostPort(opts.TCPBindAddress)
		if err != nil {
			return nil, fmt.Errorf("parsing TCP bind address (%s) failed: %w", opts.TCPBindAddress, err)
		}

		tcp := listeners.NewTCP("t1", opts.TCPBindAddress)

		var tcpAuthController auth.Controller
		if opts.TCPAuthEnabled {
			var err error
			tcpAuthController, err = NewAuthAllowUsers(opts.TCPAuthPasswordSalt, opts.TCPAuthUsers)
			if err != nil {
				return nil, fmt.Errorf("enabling TCP Authentication failed: %w", err)
			}
		} else {
			tcpAuthController = &AuthAllowEveryone{}
		}

		var tlsConfig *tls.Config
		if opts.TCPTLSEnabled {
			var err error

			tlsConfig, err = NewTLSConfig(opts.TCPTLSCertificatePath, opts.TCPTLSPrivateKeyPath)
			if err != nil {
				return nil, fmt.Errorf("enabling TCP TLS failed: %w", err)
			}
		}

		if err := server.AddListener(tcp, &listeners.Config{
			Auth:      tcpAuthController,
			TLSConfig: tlsConfig,
		}); err != nil {
			return nil, fmt.Errorf("adding TCP listener failed: %w", err)
		}
	}

	s := subscriptionmanager.New(
		subscriptionmanager.WithMaxTopicSubscriptionsPerClient[string, string](opts.MaxTopicSubscriptionsPerClient),
		subscriptionmanager.WithCleanupThresholdCount[string, string](opts.TopicCleanupThresholdCount),
		subscriptionmanager.WithCleanupThresholdRatio[string, string](opts.TopicCleanupThresholdRatio),
	)

	// this event is used to drop malicious clients
	unhook := s.Events().DropClient.Hook(func(event *subscriptionmanager.DropClientEvent[string]) {
		client, exists := server.Clients.Get(event.ClientID)
		if !exists {
			return
		}

		// stop the client connection
		client.Stop(event.Reason)

		// delete the client from the broker
		server.Clients.Delete(event.ClientID)
	}).Unhook

	// bind the broker events to the SubscriptionManager to track the subscriptions
	server.Events.OnConnect = func(cl events.Client, pk events.Packet) {
		s.Connect(cl.ID)
	}

	server.Events.OnDisconnect = func(cl events.Client, err error) {
		s.Disconnect(cl.ID)
	}

	server.Events.OnSubscribe = func(topic string, cl events.Client, qos byte) {
		s.Subscribe(cl.ID, topic)
	}

	server.Events.OnUnsubscribe = func(topic string, cl events.Client) {
		s.Unsubscribe(cl.ID, topic)
	}

	return &broker{
		server:              server,
		opts:                opts,
		subscriptionManager: s,
		unhook:              unhook,
	}, nil
}

// Events returns the events of the broker.
func (b *broker) Events() subscriptionmanager.Events[string, string] {
	return *b.subscriptionManager.Events()
}

// Start the broker.
func (b *broker) Start() error {
	return b.server.Serve()
}

// Stop the broker.
func (b *broker) Stop() error {
	if b.unhook != nil {
		b.unhook()
	}

	return b.server.Close()
}

// SystemInfo returns the metrics of the broker.
func (b *broker) HasSubscribers(topic string) bool {
	return b.subscriptionManager.TopicHasSubscribers(topic)
}

// Send publishes a message.
func (b *broker) Send(topic string, payload []byte) error {
	return b.server.Publish(topic, payload, false)
}

// SystemInfo returns the metrics of the broker.
func (b *broker) SystemInfo() *system.Info {
	return b.server.System
}

// SubscribersSize returns the size of the underlying map of the SubscriptionManager.
func (b *broker) SubscribersSize() int {
	return b.subscriptionManager.SubscribersSize()
}

// TopicsSize returns the size of all underlying maps of the SubscriptionManager.
func (b *broker) TopicsSize() int {
	return b.subscriptionManager.TopicsSize()
}
