package mqtt

import "github.com/iotaledger/hive.go/core/app"

type ParametersMQTT struct {
	BufferSize      int `default:"0" usage:"the size of the client buffers in bytes"`
	BufferBlockSize int `default:"0" usage:"the size per client buffer R/W block in bytes"`

	Subscriptions struct {
		MaxTopicSubscriptionsPerClient int     `default:"1000" usage:"the maximum number of topic subscriptions per client before the client gets dropped (DOS protection)"`
		TopicsCleanupThresholdCount    int     `default:"10000" usage:"the number of deleted topics that trigger a garbage collection of the subscription manager"`
		TopicsCleanupThresholdRatio    float32 `default:"1.0" usage:"the ratio of subscribed topics to deleted topics that trigger a garbage collection of the subscription manager"`
	}

	Websocket struct {
		Enabled     bool   `default:"true" usage:"whether to enable the websocket connection of the MQTT broker"`
		BindAddress string `default:"localhost:1888" usage:"the websocket bind address on which the MQTT broker listens on"`
	}

	TCP struct {
		Enabled     bool   `default:"false" usage:"whether to enable the TCP connection of the MQTT broker"`
		BindAddress string `default:"localhost:1883" usage:"the TCP bind address on which the MQTT broker listens on"`

		Auth struct {
			Enabled      bool              `default:"false" usage:"whether to enable auth for TCP connections"`
			PasswordSalt string            `default:"0000000000000000000000000000000000000000000000000000000000000000" usage:"the auth salt used for hashing the passwords of the users"`
			Users        map[string]string `usage:"the list of allowed users with their password+salt as a scrypt hash"`
		}

		TLS struct {
			Enabled         bool   `default:"false" usage:"whether to enable TLS for TCP connections"`
			PrivateKeyPath  string `default:"private_key.pem" usage:"the path to the private key file (x509 PEM) for TCP connections with TLS"`
			CertificatePath string `default:"certificate.pem" usage:"the path to the certificate file (x509 PEM) for TCP connections with TLS"`
		} `name:"tls"`
	} `name:"tcp"`
}

var ParamsMQTT = &ParametersMQTT{}

var params = &app.ComponentParams{
	Params: map[string]any{
		"mqtt": ParamsMQTT,
	},
	Masked: nil,
}
