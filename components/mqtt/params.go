package mqtt

import "github.com/iotaledger/hive.go/app"

type ParametersMQTT struct {
	Websocket struct {
		Enabled          bool   `default:"true" usage:"whether to enable the websocket connection of the MQTT broker"`
		BindAddress      string `default:"localhost:1888" usage:"the websocket bind address on which the MQTT broker listens on"`
		AdvertiseAddress string `default:"" usage:"the address of the websocket of the MQTT broker which is advertised to the INX Server (optional)."`
	}

	TCP struct {
		Enabled     bool   `default:"false" usage:"whether to enable the TCP connection of the MQTT broker"`
		BindAddress string `default:"localhost:1883" usage:"the TCP bind address on which the MQTT broker listens on"`

		TLS struct {
			Enabled         bool   `default:"false" usage:"whether to enable TLS for TCP connections"`
			PrivateKeyPath  string `default:"private_key.pem" usage:"the path to the private key file (x509 PEM) for TCP connections with TLS"`
			CertificatePath string `default:"certificate.pem" usage:"the path to the certificate file (x509 PEM) for TCP connections with TLS"`
		} `name:"tls"`
	} `name:"tcp"`

	Auth struct {
		PasswordSalt string            `default:"0000000000000000000000000000000000000000000000000000000000000000" usage:"the auth salt used for hashing the passwords of the users"`
		Users        map[string]string `usage:"the list of allowed users with their password+salt as a scrypt hash"`
	}

	PublicTopics    []string `usage:"the MQTT topics which can be subscribed to without authorization. Wildcards using * are allowed"`
	ProtectedTopics []string `usage:"the MQTT topics which only can be subscribed to with valid authorization. Wildcards using * are allowed"`

	Subscriptions struct {
		MaxTopicSubscriptionsPerClient int     `default:"1000" usage:"the maximum number of topic subscriptions per client before the client gets dropped (DOS protection)"`
		TopicsCleanupThresholdCount    int     `default:"10000" usage:"the number of deleted topics that trigger a garbage collection of the subscription manager"`
		TopicsCleanupThresholdRatio    float32 `default:"1.0" usage:"the ratio of subscribed topics to deleted topics that trigger a garbage collection of the subscription manager"`
	}

	MaximumClientWritesPending int `default:"8192" usage:"the maximum number of pending message writes for a client"`
	ClientWriteBufferSize      int `default:"2048" usage:"the size of the client write buffer"`
	ClientReadBufferSize       int `default:"2048" usage:"the size of the client read buffer"`
}

var ParamsMQTT = &ParametersMQTT{
	PublicTopics: []string{
		"commitments/*",
		"blocks*",
		"transactions/*",
		"block-metadata/*",
		"outputs/*",
	},
	ProtectedTopics: []string{},
}

var params = &app.ComponentParams{
	Params: map[string]any{
		"mqtt": ParamsMQTT,
	},
	Masked: []string{"mqtt.auth.passwordSalt"},
}
