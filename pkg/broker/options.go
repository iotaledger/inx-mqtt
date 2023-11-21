package broker

// Options are options around the broker.
type Options struct {
	// BufferSize is the size of the client buffers in bytes.
	BufferSize int
	// BufferBlockSize is the size per client buffer R/W block in bytes.
	BufferBlockSize int

	// MaxTopicSubscriptionsPerClient defines the maximum number of topic subscriptions per client before the client gets dropped (DOS protection).
	MaxTopicSubscriptionsPerClient int
	// TopicCleanupThresholdCount defines the number of deleted topics that trigger a garbage collection of the SubscriptionManager.
	TopicCleanupThresholdCount int
	// TopicCleanupThresholdRatio defines the ratio of subscribed topics to deleted topics that trigger a garbage collection of the SubscriptionManager.
	TopicCleanupThresholdRatio float32

	// WebsocketEnabled defines whether to enable the websocket connection of the MQTT broker.
	WebsocketEnabled bool
	// WebsocketBindAddress defines the websocket bind address on which the MQTT broker listens on.
	WebsocketBindAddress string
	// TCPEnabled defines whether to enable the TCP connection of the MQTT broker.
	TCPEnabled bool
	// TCPBindAddress defines the TCP bind address on which the MQTT broker listens on.
	TCPBindAddress string

	// TCPAuthEnabled defines whether to enable auth for TCP connections.
	TCPAuthEnabled bool
	// TCPAuthPasswordSalt is the auth salt used for hashing the passwords of the users.
	TCPAuthPasswordSalt string
	// TCPAuthUsers is the list of allowed users with their password+salt as a scrypt hash.
	TCPAuthUsers map[string]string

	// TCPTLSEnabled defines whether to enable TLS for TCP connections.
	TCPTLSEnabled bool
	// TCPTLSCertificatePath is the path to the certificate file (x509 PEM) for TCP connections with TLS.
	TCPTLSCertificatePath string
	// TCPTLSPrivateKeyPath is the path to the private key file (x509 PEM) for TCP connections with TLS.
	TCPTLSPrivateKeyPath string
}

var defaultOpts = []Option{
	WithBufferSize(0),
	WithBufferBlockSize(0),
	WithTopicCleanupThresholdCount(10000),
	WithTopicCleanupThresholdRatio(1.0),
	WithWebsocketEnabled(true),
	WithWebsocketBindAddress("localhost:1888"),
	WithTCPEnabled(false),
	WithTCPBindAddress("localhost:1883"),
	WithTCPAuthEnabled(false),
	WithTCPAuthPasswordSalt("0000000000000000000000000000000000000000000000000000000000000000"),
	WithTCPAuthUsers(map[string]string{}),
	WithTCPTLSEnabled(false),
	WithTCPTLSCertificatePath(""),
	WithTCPTLSPrivateKeyPath(""),
}

// applies the given Option.
func (bo *Options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(bo)
	}
}

// ApplyOnDefault applies the given options on top of the default options.
func (bo *Options) ApplyOnDefault(opts ...Option) {
	bo.apply(defaultOpts...)
	bo.apply(opts...)
}

// Option is a function which sets an option on a Options instance.
type Option func(options *Options)

// WithBufferSize sets the size of the client buffers in bytes.
func WithBufferSize(bufferSize int) Option {
	return func(options *Options) {
		options.BufferSize = bufferSize
	}
}

// WithBufferBlockSize sets the size per client buffer R/W block in bytes.
func WithBufferBlockSize(bufferBlockSize int) Option {
	return func(options *Options) {
		options.BufferBlockSize = bufferBlockSize
	}
}

// WithMaxTopicSubscriptionsPerClient sets the maximum number of topic subscriptions per client before the client gets dropped (DOS protection).
func WithMaxTopicSubscriptionsPerClient(maxTopicSubscriptionsPerClient int) Option {
	return func(options *Options) {
		options.MaxTopicSubscriptionsPerClient = maxTopicSubscriptionsPerClient
	}
}

// WithTopicCleanupThresholdCount sets the number of deleted topics that trigger a garbage collection of the SubscriptionManager.
func WithTopicCleanupThresholdCount(topicCleanupThresholdCount int) Option {
	return func(options *Options) {
		options.TopicCleanupThresholdCount = topicCleanupThresholdCount
	}
}

// WithTopicCleanupThresholdRatio the ratio of subscribed topics to deleted topics that trigger a garbage collection of the SubscriptionManager.
func WithTopicCleanupThresholdRatio(topicCleanupThresholdRatio float32) Option {
	return func(options *Options) {
		options.TopicCleanupThresholdRatio = topicCleanupThresholdRatio
	}
}

// WithWebsocketEnabled sets whether to enable the websocket connection of the MQTT broker.
func WithWebsocketEnabled(websocketEnabled bool) Option {
	return func(options *Options) {
		options.WebsocketEnabled = websocketEnabled
	}
}

// WithWebsocketBindAddress sets the websocket bind address on which the MQTT broker listens on.
func WithWebsocketBindAddress(websocketBindAddress string) Option {
	return func(options *Options) {
		options.WebsocketBindAddress = websocketBindAddress
	}
}

// WithTCPEnabled sets whether to enable the TCP connection of the MQTT broker.
func WithTCPEnabled(tcpEnabled bool) Option {
	return func(options *Options) {
		options.TCPEnabled = tcpEnabled
	}
}

// WithTCPBindAddress sets the TCP bind address on which the MQTT broker listens on.
func WithTCPBindAddress(tcpBindAddress string) Option {
	return func(options *Options) {
		options.TCPBindAddress = tcpBindAddress
	}
}

// WithTCPAuthEnabled sets whether to enable auth for TCP connections.
func WithTCPAuthEnabled(tcpAuthEnabled bool) Option {
	return func(options *Options) {
		options.TCPAuthEnabled = tcpAuthEnabled
	}
}

// WithTCPAuthPasswordSalt sets the auth salt used for hashing the passwords of the users.
func WithTCPAuthPasswordSalt(tcpAuthPasswordSalt string) Option {
	return func(options *Options) {
		options.TCPAuthPasswordSalt = tcpAuthPasswordSalt
	}
}

// WithTCPAuthUsers sets the list of allowed users with their password+salt as a scrypt hash.
func WithTCPAuthUsers(tcpAuthUsers map[string]string) Option {
	return func(options *Options) {
		options.TCPAuthUsers = tcpAuthUsers
	}
}

// WithTCPTLSEnabled sets whether to enable TLS for TCP connections.
func WithTCPTLSEnabled(tcpTLSEnabled bool) Option {
	return func(options *Options) {
		options.TCPTLSEnabled = tcpTLSEnabled
	}
}

// WithTCPTLSCertificatePath sets the path to the certificate file (x509 PEM) for TCP connections with TLS.
func WithTCPTLSCertificatePath(tcpTLSCertificatePath string) Option {
	return func(options *Options) {
		options.TCPTLSCertificatePath = tcpTLSCertificatePath
	}
}

// WithTCPTLSPrivateKeyPath sets the path to the private key file (x509 PEM) for TCP connections with TLS.
func WithTCPTLSPrivateKeyPath(tcpTLSPrivateKeyPath string) Option {
	return func(options *Options) {
		options.TCPTLSPrivateKeyPath = tcpTLSPrivateKeyPath
	}
}
