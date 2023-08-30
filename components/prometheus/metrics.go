package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/iotaledger/inx-mqtt/components/mqtt"
)

var (
	mqttBrokerAppInfo                            *prometheus.GaugeVec
	mqttBrokerStarted                            prometheus.Gauge
	mqttBrokerUptime                             prometheus.Gauge
	mqttBrokerBytesRecv                          prometheus.Gauge
	mqttBrokerBytesSent                          prometheus.Gauge
	mqttBrokerClientsConnected                   prometheus.Gauge
	mqttBrokerClientsDisconnected                prometheus.Gauge
	mqttBrokerClientsMax                         prometheus.Gauge
	mqttBrokerClientsTotal                       prometheus.Gauge
	mqttBrokerConnectionsTotal                   prometheus.Gauge
	mqttBrokerMessagesRecv                       prometheus.Gauge
	mqttBrokerMessagesSent                       prometheus.Gauge
	mqttBrokerPublishDropped                     prometheus.Gauge
	mqttBrokerPublishRecv                        prometheus.Gauge
	mqttBrokerPublishSent                        prometheus.Gauge
	mqttBrokerRetained                           prometheus.Gauge
	mqttBrokerInflight                           prometheus.Gauge
	mqttBrokerSubscriptions                      prometheus.Gauge
	mqttBrokerSubscriptionManagerSubscribersSize prometheus.Gauge
	mqttBrokerSubscriptionManagerTopicsSize      prometheus.Gauge
)

func registerNewMQTTBrokerGaugeVec(registry *prometheus.Registry, name string, labelNames []string, help string) *prometheus.GaugeVec {
	gaugeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "iota",
			Subsystem: "mqtt_broker",
			Name:      name,
			Help:      help,
		}, labelNames)
	registry.MustRegister(gaugeVec)

	return gaugeVec
}

func registerNewMQTTBrokerGauge(registry *prometheus.Registry, name string, help string) prometheus.Gauge {
	gauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "iota",
			Subsystem: "mqtt_broker",
			Name:      name,
			Help:      help,
		})
	registry.MustRegister(gauge)

	return gauge
}

func registerMQTTMetrics(registry *prometheus.Registry) {
	mqttBrokerAppInfo = registerNewMQTTBrokerGaugeVec(registry, "app_info", []string{"name", "version", "broker_version"}, "The current version of the server.")
	mqttBrokerStarted = registerNewMQTTBrokerGauge(registry, "started", "The time the server started in unix seconds.")
	mqttBrokerUptime = registerNewMQTTBrokerGauge(registry, "uptime", "The number of seconds the server has been online.")
	mqttBrokerBytesRecv = registerNewMQTTBrokerGauge(registry, "bytes_recv", "The total number of bytes received in all packets.")
	mqttBrokerBytesSent = registerNewMQTTBrokerGauge(registry, "bytes_sent", "The total number of bytes sent to clients.")
	mqttBrokerClientsConnected = registerNewMQTTBrokerGauge(registry, "clients_connected", "The number of currently connected clients.")
	mqttBrokerClientsDisconnected = registerNewMQTTBrokerGauge(registry, "clients_disconnected", "The number of disconnected non-cleansession clients.")
	mqttBrokerClientsMax = registerNewMQTTBrokerGauge(registry, "clients_max", "The maximum number of clients that have been concurrently connected.")
	mqttBrokerClientsTotal = registerNewMQTTBrokerGauge(registry, "clients_total", "The sum of all clients, connected and disconnected.")
	mqttBrokerConnectionsTotal = registerNewMQTTBrokerGauge(registry, "connections_total", "The sum number of clients which have ever connected.")
	mqttBrokerMessagesRecv = registerNewMQTTBrokerGauge(registry, "messages_recv", "The total number of packets received.")
	mqttBrokerMessagesSent = registerNewMQTTBrokerGauge(registry, "messages_sent", "The total number of packets sent.")
	mqttBrokerPublishDropped = registerNewMQTTBrokerGauge(registry, "publish_dropped", "The number of in-flight publish messages which were dropped.")
	mqttBrokerPublishRecv = registerNewMQTTBrokerGauge(registry, "publish_recv", "The total number of received publish packets.")
	mqttBrokerPublishSent = registerNewMQTTBrokerGauge(registry, "publish_sent", "The total number of sent publish packets.")
	mqttBrokerRetained = registerNewMQTTBrokerGauge(registry, "retained", "The number of messages currently retained.")
	mqttBrokerInflight = registerNewMQTTBrokerGauge(registry, "inflight", "The number of messages currently in-flight.")
	mqttBrokerSubscriptions = registerNewMQTTBrokerGauge(registry, "subscriptions", "The total number of filter subscriptions.")
	mqttBrokerSubscriptionManagerSubscribersSize = registerNewMQTTBrokerGauge(registry, "subscription_manager_subscribers_size", "The number of active subscribers in the subscription manager.")
	mqttBrokerSubscriptionManagerTopicsSize = registerNewMQTTBrokerGauge(registry, "subscription_manager_topics_size", "The number of active topics in the subscription manager.")
}

func collectMQTTBroker(server *mqtt.Server) {
	mqttBrokerAppInfo.With(prometheus.Labels{
		"name":           Component.App().Info().Name,
		"version":        Component.App().Info().Version,
		"broker_version": server.MQTTBroker.SystemInfo().Version,
	}).Set(1)
	mqttBrokerStarted.Set(float64(server.MQTTBroker.SystemInfo().Started))
	mqttBrokerUptime.Set(float64(server.MQTTBroker.SystemInfo().Uptime))
	mqttBrokerBytesRecv.Set(float64(server.MQTTBroker.SystemInfo().BytesRecv))
	mqttBrokerBytesSent.Set(float64(server.MQTTBroker.SystemInfo().BytesSent))
	mqttBrokerClientsConnected.Set(float64(server.MQTTBroker.SystemInfo().ClientsConnected))
	mqttBrokerClientsDisconnected.Set(float64(server.MQTTBroker.SystemInfo().ClientsDisconnected))
	mqttBrokerClientsMax.Set(float64(server.MQTTBroker.SystemInfo().ClientsMax))
	mqttBrokerClientsTotal.Set(float64(server.MQTTBroker.SystemInfo().ClientsTotal))
	mqttBrokerConnectionsTotal.Set(float64(server.MQTTBroker.SystemInfo().ConnectionsTotal))
	mqttBrokerMessagesRecv.Set(float64(server.MQTTBroker.SystemInfo().MessagesRecv))
	mqttBrokerMessagesSent.Set(float64(server.MQTTBroker.SystemInfo().MessagesSent))
	mqttBrokerPublishDropped.Set(float64(server.MQTTBroker.SystemInfo().PublishDropped))
	mqttBrokerPublishRecv.Set(float64(server.MQTTBroker.SystemInfo().PublishRecv))
	mqttBrokerPublishSent.Set(float64(server.MQTTBroker.SystemInfo().PublishSent))
	mqttBrokerRetained.Set(float64(server.MQTTBroker.SystemInfo().Retained))
	mqttBrokerInflight.Set(float64(server.MQTTBroker.SystemInfo().Inflight))
	mqttBrokerSubscriptions.Set(float64(server.MQTTBroker.SystemInfo().Subscriptions))
	mqttBrokerSubscriptionManagerSubscribersSize.Set(float64(server.MQTTBroker.SubscribersSize()))
	mqttBrokerSubscriptionManagerTopicsSize.Set(float64(server.MQTTBroker.TopicsSize()))
}
