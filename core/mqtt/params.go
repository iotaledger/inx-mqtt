package mqtt

import (
	flag "github.com/spf13/pflag"
)

const (
	// CfgINXAddress the INX address to which to connect to.
	CfgINXAddress = "inx.address"

	// CfgMQTTBufferSize is the size of the client buffers in bytes.
	CfgMQTTBufferSize = "mqtt.bufferSize"
	// CfgMQTTBufferBlockSize is the size per client buffer R/W block in bytes.
	CfgMQTTBufferBlockSize = "mqtt.bufferBlockSize"
	// CfgMQTTTopicCleanupThreshold the number of deleted topics that trigger a garbage collection of the topic manager.
	CfgMQTTTopicCleanupThreshold = "mqtt.topicCleanupThreshold"

	// CfgMQTTWebsocketEnabled defines whether to enable the websocket connection of the MQTT broker.
	CfgMQTTWebsocketEnabled = "mqtt.websocket.enabled"
	// CfgMQTTWebsocketBindAddress the websocket bind address on which the MQTT broker listens on.
	CfgMQTTWebsocketBindAddress = "mqtt.websocket.bindAddress"

	// CfgMQTTTCPEnabled defines whether to enable the TCP connection of the MQTT broker.
	CfgMQTTTCPEnabled = "mqtt.tcp.enabled"
	// CfgMQTTTCPBindAddress the TCP bind address on which the MQTT broker listens on.
	CfgMQTTTCPBindAddress = "mqtt.tcp.bindAddress"

	// CfgMQTTTCPAuthEnabled defines whether to enable auth for TCP connections.
	CfgMQTTTCPAuthEnabled = "mqtt.tcp.auth.enabled"
	// CfgMQTTTCPAuthPasswordSalt is the auth salt used for hashing the passwords of the users.
	CfgMQTTTCPAuthPasswordSalt = "mqtt.tcp.auth.passwordSalt"
	// CfgMQTTTCPAuthUsers is the list of allowed users with their password+salt as a scrypt hash.
	CfgMQTTTCPAuthUsers = "mqtt.tcp.auth.users"

	// CfgMQTTTCPTLSEnabled defines whether to enable TLS for TCP connections.
	CfgMQTTTCPTLSEnabled = "mqtt.tcp.tls.enabled"
	// CfgMQTTTCPTLSCertificatePath is the path to the certificate file (x509 PEM) for TCP connections with TLS.
	CfgMQTTTCPTLSCertificatePath = "mqtt.tcp.tls.certificatePath"
	// CfgMQTTTCPTLSPrivateKeyPath is the path to the private key file (x509 PEM) for TCP connections with TLS.
	CfgMQTTTCPTLSPrivateKeyPath = "mqtt.tcp.tls.privateKeyPath"

	// CfgPrometheusEnabled defines whether to enable the prometheus metrics.
	CfgPrometheusEnabled = "prometheus.enabled"
	// CfgPrometheusGoMetrics defines whether to include go metrics.
	CfgPrometheusGoMetrics = "prometheus.goMetrics"
	// CfgPrometheusProcessMetrics defines whether to include process metrics.
	CfgPrometheusProcessMetrics = "prometheus.processMetrics"
	// CfgPrometheusBindAddress bind address on which the Prometheus HTTP server listens.
	CfgPrometheusBindAddress = "prometheus.bindAddress"
)

func flagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.String(CfgINXAddress, "localhost:9029", "the INX address to which to connect to")

	fs.Int(CfgMQTTBufferSize, 0, "the size of the client buffers in bytes")
	fs.Int(CfgMQTTBufferBlockSize, 0, "the size per client buffer R/W block in bytes")
	fs.Int(CfgMQTTTopicCleanupThreshold, 10000, "the number of deleted topics that trigger a garbage collection of the topic manager")

	fs.Bool(CfgMQTTWebsocketEnabled, true, "whether to enable the websocket connection of the MQTT broker")
	fs.String(CfgMQTTWebsocketBindAddress, "localhost:1888", "the websocket bind address on which the MQTT broker listens on")

	fs.Bool(CfgMQTTTCPEnabled, false, "whether to enable the TCP connection of the MQTT broker")
	fs.String(CfgMQTTTCPBindAddress, "localhost:1883", "the TCP bind address on which the MQTT broker listens on")

	fs.Bool(CfgMQTTTCPAuthEnabled, false, "whether to enable auth for TCP connections")
	fs.String(CfgMQTTTCPAuthPasswordSalt, "0000000000000000000000000000000000000000000000000000000000000000", "the auth salt used for hashing the passwords of the users")
	fs.StringToString(CfgMQTTTCPAuthUsers, map[string]string{}, "the list of allowed users with their password+salt as a scrypt hash")

	fs.Bool(CfgMQTTTCPTLSEnabled, false, "whether to enable TLS for TCP connections")
	fs.String(CfgMQTTTCPTLSCertificatePath, "", "the path to the certificate file (x509 PEM) for TCP connections with TLS")
	fs.String(CfgMQTTTCPTLSPrivateKeyPath, "", "the path to the private key file (x509 PEM) for TCP connections with TLS")

	fs.Bool(CfgPrometheusEnabled, false, "whether to enable the prometheus metrics")
	fs.Bool(CfgPrometheusGoMetrics, false, "whether to include go metrics")
	fs.Bool(CfgPrometheusProcessMetrics, false, "whether to include process metrics")
	fs.String(CfgPrometheusBindAddress, "localhost:9312", "bind address on which the Prometheus HTTP server listens.")
	return fs
}
