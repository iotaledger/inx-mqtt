package broker

// Options are options around the broker.
type Options struct {
	// WebsocketEnabled defines whether to enable the websocket connection of the MQTT broker.
	WebsocketEnabled bool
	// WebsocketBindAddress defines the websocket bind address on which the MQTT broker listens on.
	WebsocketBindAddress string

	// TCPEnabled defines whether to enable the TCP connection of the MQTT broker.
	TCPEnabled bool
	// TCPBindAddress defines the TCP bind address on which the MQTT broker listens on.
	TCPBindAddress string
	// TCPTLSEnabled defines whether to enable TLS for TCP connections.
	TCPTLSEnabled bool
	// TCPTLSCertificatePath is the path to the certificate file (x509 PEM) for TCP connections with TLS.
	TCPTLSCertificatePath string
	// TCPTLSPrivateKeyPath is the path to the private key file (x509 PEM) for TCP connections with TLS.
	TCPTLSPrivateKeyPath string

	// AuthPasswordSalt is the auth salt used for hashing the passwords of the users.
	AuthPasswordSalt string
	// AuthUsers is the list of allowed users with their password+salt as a scrypt hash.
	AuthUsers map[string]string

	// PublicTopics are the MQTT topics which can be subscribed to without authorization. Wildcards using * are allowed
	PublicTopics []string
	// ProtectedTopics are the MQTT topics which only can be subscribed to with valid authorization. Wildcards using * are allowed
	ProtectedTopics []string

	// MaxTopicSubscriptionsPerClient defines the maximum number of topic subscriptions per client before the client gets dropped (DOS protection).
	MaxTopicSubscriptionsPerClient int
	// TopicCleanupThresholdCount defines the number of deleted topics that trigger a garbage collection of the SubscriptionManager.
	TopicCleanupThresholdCount int
	// TopicCleanupThresholdRatio defines the ratio of subscribed topics to deleted topics that trigger a garbage collection of the SubscriptionManager.
	TopicCleanupThresholdRatio float32

	// MaximumClientWritesPending specifies the maximum number of pending message writes for a client.
	MaximumClientWritesPending int
	// ClientWriteBufferSize specifies the size of the client write buffer.
	ClientWriteBufferSize int
	// ClientNetReadBufferSize specifies the size of the client read buffer.
	ClientReadBufferSize int
}

var defaultOpts = []Option{
	WithWebsocketEnabled(true),
	WithWebsocketBindAddress("localhost:1888"),
	WithTCPEnabled(false),
	WithTCPBindAddress("localhost:1883"),
	WithTCPTLSEnabled(false),
	WithTCPTLSCertificatePath(""),
	WithTCPTLSPrivateKeyPath(""),
	WithAuthPasswordSalt("0000000000000000000000000000000000000000000000000000000000000000"),
	WithAuthUsers(map[string]string{}),
	WithPublicTopics([]string{"*"}),
	WithProtectedTopics([]string{}),
	WithMaxTopicSubscriptionsPerClient(1000),
	WithTopicCleanupThresholdCount(10000),
	WithTopicCleanupThresholdRatio(1.0),
	WithMaximumClientWritesPending(1024 * 8),
	WithClientWriteBufferSize(1024 * 2),
	WithClientReadBufferSize(1024 * 2),
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

// WithAuthPasswordSalt sets the auth salt used for hashing the passwords of the users.
func WithAuthPasswordSalt(tcpAuthPasswordSalt string) Option {
	return func(options *Options) {
		options.AuthPasswordSalt = tcpAuthPasswordSalt
	}
}

// WithAuthUsers sets the list of allowed users with their password+salt as a scrypt hash.
func WithAuthUsers(tcpAuthUsers map[string]string) Option {
	return func(options *Options) {
		options.AuthUsers = tcpAuthUsers
	}
}

// WithPublicTopics sets the MQTT topics which can be subscribed to without authorization. Wildcards using * are allowed.
func WithPublicTopics(publicTopics []string) Option {
	return func(options *Options) {
		options.PublicTopics = publicTopics
	}
}

// WithProtectedTopics sets the MQTT topics which only can be subscribed to with valid authorization. Wildcards using * are allowed.
func WithProtectedTopics(protectedTopics []string) Option {
	return func(options *Options) {
		options.ProtectedTopics = protectedTopics
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

// WithMaximumClientWritesPending specifies the maximum number of pending message writes for a client.
func WithMaximumClientWritesPending(maximumClientWritesPending int) Option {
	return func(options *Options) {
		options.MaximumClientWritesPending = maximumClientWritesPending
	}
}

// WithClientWriteBufferSize specifies the size of the client write buffer.
func WithClientWriteBufferSize(clientWriteBufferSize int) Option {
	return func(options *Options) {
		options.ClientWriteBufferSize = clientWriteBufferSize
	}
}

// WithClientReadBufferSize specifies the size of the client read buffer.
func WithClientReadBufferSize(clientReadBufferSize int) Option {
	return func(options *Options) {
		options.ClientReadBufferSize = clientReadBufferSize
	}
}
