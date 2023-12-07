package broker

import (
	"crypto/tls"
	"math"
	"net"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/system"

	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/web/basicauth"
	"github.com/iotaledger/hive.go/web/subscriptionmanager"
)

type Broker interface {
	// Events returns the events of the broker.
	Events() *subscriptionmanager.Events[string, string]
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
		return nil, ierrors.New("at least websocket or TCP must be enabled")
	}

	server := mqtt.New(&mqtt.Options{
		// Capabilities defines the server features and behavior.
		Capabilities: &mqtt.Capabilities{
			MaximumSessionExpiryInterval: 0,              // we don't keep disconnected sessions
			MaximumMessageExpiryInterval: 10,             // maximum message expiry if message expiry is 0 or over
			ReceiveMaximum:               1024,           // maximum number of concurrent qos messages per client
			MaximumQos:                   0,              // we don't support QoS 1 and 2, we only "fire and forget"
			RetainAvailable:              0,              // retain messages is disabled in our usecase, we handle it ourselves
			MaximumPacketSize:            1024,           // we allow some bytes for the CONNECT and SUBSCRIBE messages, but otherwise the client is not allowed to send any messages anyways
			TopicAliasMaximum:            math.MaxUint16, // maximum topic alias value (default)
			WildcardSubAvailable:         0,              // wildcard subscriptions are disabled because we handle them ourselves, not on server side
			SubIDAvailable:               0,              // subscription identifiers are disabled, since we don't enable wildcard subscriptions in the server
			SharedSubAvailable:           0,              // shared subscriptions are prohibited in our usecase
			MinimumProtocolVersion:       3,              // minimum supported mqtt version (3.0.0) (default)
			MaximumClientWritesPending:   int32(opts.MaximumClientWritesPending),
		},
		ClientNetWriteBufferSize: opts.ClientWriteBufferSize,
		ClientNetReadBufferSize:  opts.ClientReadBufferSize,
		InlineClient:             true, // this needs to be set to true to allow the broker to send messages itself
	})

	if opts.WebsocketEnabled {
		// check websocket bind address
		_, _, err := net.SplitHostPort(opts.WebsocketBindAddress)
		if err != nil {
			return nil, ierrors.Errorf("parsing websocket bind address (%s) failed: %w", opts.WebsocketBindAddress, err)
		}

		// skip TLS config since it is added via traefik in our recommended setup anyway
		ws := listeners.NewWebsocket("ws1", opts.WebsocketBindAddress, &listeners.Config{TLSConfig: nil})

		if err := server.AddListener(ws); err != nil {
			return nil, ierrors.Errorf("adding websocket listener failed: %w", err)
		}
	}

	if opts.TCPEnabled {
		// check tcp bind address
		_, _, err := net.SplitHostPort(opts.TCPBindAddress)
		if err != nil {
			return nil, ierrors.Errorf("parsing TCP bind address (%s) failed: %w", opts.TCPBindAddress, err)
		}

		var tlsConfig *tls.Config
		if opts.TCPTLSEnabled {
			tlsConfig, err = NewTLSConfig(opts.TCPTLSCertificatePath, opts.TCPTLSPrivateKeyPath)
			if err != nil {
				return nil, ierrors.Errorf("enabling TCP TLS failed: %w", err)
			}
		}

		tcp := listeners.NewTCP("t1", opts.TCPBindAddress, &listeners.Config{
			TLSConfig: tlsConfig,
		})

		if err := server.AddListener(tcp); err != nil {
			return nil, ierrors.Errorf("adding TCP listener failed: %w", err)
		}
	}

	subscriptionManager := subscriptionmanager.New(
		subscriptionmanager.WithMaxTopicSubscriptionsPerClient[string, string](opts.MaxTopicSubscriptionsPerClient),
		subscriptionmanager.WithCleanupThresholdCount[string, string](opts.TopicCleanupThresholdCount),
		subscriptionmanager.WithCleanupThresholdRatio[string, string](opts.TopicCleanupThresholdRatio),
	)

	// this event is used to drop malicious clients
	unhook := subscriptionManager.Events().DropClient.Hook(func(event *subscriptionmanager.DropClientEvent[string]) {
		client, exists := server.Clients.Get(event.ClientID)
		if !exists {
			return
		}

		// stop the client connection
		client.Stop(event.Reason)

		// delete the client from the broker
		server.Clients.Delete(event.ClientID)
	}).Unhook

	basicAuthManager, err := basicauth.NewBasicAuthManager(opts.AuthUsers, opts.AuthPasswordSalt)
	if err != nil {
		return nil, ierrors.Errorf("failed to create basic auth manager: %w", err)
	}

	// bind the broker events to the SubscriptionManager to track the subscriptions
	brokerHook, err := NewBrokerHook(subscriptionManager, basicAuthManager, opts.PublicTopics, opts.ProtectedTopics)
	if err != nil {
		return nil, ierrors.Errorf("failed to create broker hook: %w", err)
	}

	if err := server.AddHook(brokerHook, nil); err != nil {
		return nil, ierrors.Errorf("failed to add broker hook for subscriptions: %w", err)
	}

	return &broker{
		server:              server,
		opts:                opts,
		subscriptionManager: subscriptionManager,
		unhook:              unhook,
	}, nil
}

// Events returns the events of the broker.
func (b *broker) Events() *subscriptionmanager.Events[string, string] {
	return b.subscriptionManager.Events()
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
	// we publish all messages with QoS 0 ("fire and forget")
	return b.server.Publish(topic, payload, false, 0)
}

// SystemInfo returns the metrics of the broker.
func (b *broker) SystemInfo() *system.Info {
	return b.server.Info.Clone()
}

// SubscribersSize returns the size of the underlying map of the SubscriptionManager.
func (b *broker) SubscribersSize() int {
	return b.subscriptionManager.SubscribersSize()
}

// TopicsSize returns the size of all underlying maps of the SubscriptionManager.
func (b *broker) TopicsSize() int {
	return b.subscriptionManager.TopicsSize()
}
