package mqtt

// Options are options around the MQTT Server.
type Options struct {
	// WebsocketEnabled defines whether to enable the websocket connection of the MQTT broker.
	WebsocketEnabled bool
	// WebsocketBindAddress defines the websocket bind address on which the MQTT broker listens on.
	WebsocketBindAddress string
	// WebsocketAdvertiseAddress defines the address of the websocket of the MQTT broker which is advertised to the INX Server (optional).
	WebsocketAdvertiseAddress string
}

var defaultOpts = []Option{
	WithWebsocketEnabled(true),
	WithWebsocketBindAddress("localhost:1888"),
	WithWebsocketAdvertiseAddress(""),
}

// applies the given Option.
func (so *Options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(so)
	}
}

// ApplyOnDefault applies the given options on top of the default options.
func (so *Options) ApplyOnDefault(opts ...Option) {
	so.apply(defaultOpts...)
	so.apply(opts...)
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

// WithWebsocketBindAddress sets the address of the websocket of the MQTT broker which is advertised to the INX Server (optional).
func WithWebsocketAdvertiseAddress(websocketAdvertiseAddress string) Option {
	return func(options *Options) {
		options.WebsocketAdvertiseAddress = websocketAdvertiseAddress
	}
}
