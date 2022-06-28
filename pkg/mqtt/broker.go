package mqtt

import (
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
	broker      *mqtt.Server
	opts        *BrokerOptions
	subMananger *subscriberManager
}

// NewBroker creates a new broker.
func NewBroker(onConnect OnConnectH, onDisconnect OnDisconnectH, onSubscribe OnSubscribeH, onUnsubscribe OnUnsubscribeH, brokerOpts *BrokerOptions) (*Broker, error) {

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
				return nil, fmt.Errorf("Enabling TCP Authentication failed: %w", err)
			}
		} else {
			tcpAuthController = &AuthAllowEveryone{}
		}

		var tls *listeners.TLS
		if brokerOpts.TCPTLSEnabled {
			var err error
			tls, err = NewTLSSettings(brokerOpts.TCPTLSCertificatePath, brokerOpts.TCPTLSPrivateKeyPath)
			if err != nil {
				return nil, fmt.Errorf("Enabling TCP TLS failed: %w", err)
			}
		}

		if err := broker.AddListener(tcp, &listeners.Config{
			Auth: tcpAuthController,
			TLS:  tls,
		}); err != nil {
			return nil, fmt.Errorf("adding TCP listener failed: %w", err)
		}
	}

	s := newSubscriberManager(onConnect, onDisconnect, onSubscribe, onUnsubscribe, brokerOpts.TopicCleanupThreshold)
	// bind the broker events to the topic manager to track the subscriptions
	broker.Events.OnSubscribe = func(filter string, cl events.Client, qos byte) {
		fmt.Printf("<< OnSubscribe %s: %v\n", cl.ID, filter)
		s.Subscribe(cl.ID, filter)
	}

	broker.Events.OnUnsubscribe = func(filter string, cl events.Client) {
		fmt.Printf("<< OnUnsubscribe %s: %v\n", cl.ID, filter)
		s.Unsubscribe(cl.ID, filter)
	}

	broker.Events.OnConnect = func(cl events.Client, pk events.Packet) {
		fmt.Printf("<< OnConnect client connected %s: \n", cl.ID)
		s.Connect(cl.ID)
	}

	broker.Events.OnDisconnect = func(cl events.Client, err error) {
		fmt.Printf("<< OnDisconnect client disconnected %s: %v\n", cl.ID, err)
		s.Disconnect(cl.ID)
	}

	return &Broker{
		broker:      broker,
		opts:        brokerOpts,
		subMananger: s,
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
	return b.subMananger.hasTopic(topic)
}

// Send publishes a message.
func (b *Broker) Send(topic string, payload []byte) error {
	return b.broker.Publish(topic, payload, false)
}

// TopicsManagerSize returns the size of the underlying map of the topics manager.
func (b *Broker) TopicsManagerSize() int {
	return b.subMananger.Size()
}
