package mqtt

// BrokerOptions are options around the broker.
type BrokerOptions struct {
	// BufferSize is the size of the client buffers in bytes.
	BufferSize int
	// BufferBlockSize is the size per client buffer R/W block in bytes.
	BufferBlockSize int
	// TopicCleanupThreshold the number of deleted topics that trigger a garbage collection of the topic manager.
	TopicCleanupThreshold int

	// WebsocketEnabled defines whether to enable the websocket connection of the MQTT broker.
	WebsocketEnabled bool
	// WebsocketBindAddress the websocket bind address on which the MQTT broker listens on.
	WebsocketBindAddress string

	// TCPEnabled defines whether to enable the TCP connection of the MQTT broker.
	TCPEnabled bool
	// TCPBindAddress the TCP bind address on which the MQTT broker listens on.
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

var defaultBrokerOpts = []BrokerOption{
	WithBufferSize(0),
	WithBufferBlockSize(0),
	WithTopicCleanupThreshold(10000),
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

// applies the given BrokerOption.
func (bo *BrokerOptions) apply(opts ...BrokerOption) {
	for _, opt := range opts {
		opt(bo)
	}
}

// ApplyOnDefault applies the given options on top of the default options.
func (bo *BrokerOptions) ApplyOnDefault(opts ...BrokerOption) {
	bo.apply(defaultBrokerOpts...)
	bo.apply(opts...)
}

// BrokerOption is a function which sets an option on a BrokerOptions instance.
type BrokerOption func(options *BrokerOptions)

// WithBufferSize sets the size of the client buffers in bytes.
func WithBufferSize(bufferSize int) BrokerOption {
	return func(options *BrokerOptions) {
		options.BufferSize = bufferSize
	}
}

// WithBufferBlockSize sets the size per client buffer R/W block in bytes.
func WithBufferBlockSize(bufferBlockSize int) BrokerOption {
	return func(options *BrokerOptions) {
		options.BufferBlockSize = bufferBlockSize
	}
}

// WithTopicCleanupThreshold sets the number of deleted topics that trigger a garbage collection of the topic manager.
func WithTopicCleanupThreshold(topicCleanupThreshold int) BrokerOption {
	return func(options *BrokerOptions) {
		options.TopicCleanupThreshold = topicCleanupThreshold
	}
}

// WithWebsocketEnabled sets whether to enable the websocket connection of the MQTT broker.
func WithWebsocketEnabled(websocketEnabled bool) BrokerOption {
	return func(options *BrokerOptions) {
		options.WebsocketEnabled = websocketEnabled
	}
}

// WithWebsocketBindAddress sets the websocket bind address on which the MQTT broker listens on.
func WithWebsocketBindAddress(websocketBindAddress string) BrokerOption {
	return func(options *BrokerOptions) {
		options.WebsocketBindAddress = websocketBindAddress
	}
}

// WithTCPEnabled sets whether to enable the TCP connection of the MQTT broker.
func WithTCPEnabled(tcpEnabled bool) BrokerOption {
	return func(options *BrokerOptions) {
		options.TCPEnabled = tcpEnabled
	}
}

// WithTCPBindAddress sets the TCP bind address on which the MQTT broker listens on.
func WithTCPBindAddress(tcpBindAddress string) BrokerOption {
	return func(options *BrokerOptions) {
		options.TCPBindAddress = tcpBindAddress
	}
}

// WithTCPAuthEnabled sets whether to enable auth for TCP connections.
func WithTCPAuthEnabled(tcpAuthEnabled bool) BrokerOption {
	return func(options *BrokerOptions) {
		options.TCPAuthEnabled = tcpAuthEnabled
	}
}

// WithTCPAuthPasswordSalt sets the auth salt used for hashing the passwords of the users.
func WithTCPAuthPasswordSalt(tcpAuthPasswordSalt string) BrokerOption {
	return func(options *BrokerOptions) {
		options.TCPAuthPasswordSalt = tcpAuthPasswordSalt
	}
}

// WithTCPAuthUsers sets the list of allowed users with their password+salt as a scrypt hash.
func WithTCPAuthUsers(tcpAuthUsers map[string]string) BrokerOption {
	return func(options *BrokerOptions) {
		options.TCPAuthUsers = tcpAuthUsers
	}
}

// WithTCPTLSEnabled sets whether to enable TLS for TCP connections.
func WithTCPTLSEnabled(tcpTlsEnabled bool) BrokerOption {
	return func(options *BrokerOptions) {
		options.TCPTLSEnabled = tcpTlsEnabled
	}
}

// WithTCPTLSCertificatePath sets the path to the certificate file (x509 PEM) for TCP connections with TLS.
func WithTCPTLSCertificatePath(tcpTlsCertificatePath string) BrokerOption {
	return func(options *BrokerOptions) {
		options.TCPTLSCertificatePath = tcpTlsCertificatePath
	}
}

// WithTCPTLSPrivateKeyPath sets the path to the private key file (x509 PEM) for TCP connections with TLS.
func WithTCPTLSPrivateKeyPath(tcpTlsPrivateKeyPath string) BrokerOption {
	return func(options *BrokerOptions) {
		options.TCPTLSPrivateKeyPath = tcpTlsPrivateKeyPath
	}
}
