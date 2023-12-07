package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/iotaledger/inx-mqtt/pkg/mqtt"
)

var (
	mqttBrokerAppInfo                            *prometheus.GaugeVec
	mqttBrokerStarted                            prometheus.Gauge
	mqttBrokerTime                               prometheus.Gauge
	mqttBrokerUptime                             prometheus.Gauge
	mqttBrokerBytesReceived                      prometheus.Gauge
	mqttBrokerBytesSent                          prometheus.Gauge
	mqttBrokerClientsConnected                   prometheus.Gauge
	mqttBrokerClientsDisconnected                prometheus.Gauge
	mqttBrokerClientsMaximum                     prometheus.Gauge
	mqttBrokerClientsTotal                       prometheus.Gauge
	mqttBrokerMessagesReceived                   prometheus.Gauge
	mqttBrokerMessagesSent                       prometheus.Gauge
	mqttBrokerMessagesDropped                    prometheus.Gauge
	mqttBrokerRetained                           prometheus.Gauge
	mqttBrokerInflight                           prometheus.Gauge
	mqttBrokerInflightDropped                    prometheus.Gauge
	mqttBrokerSubscriptions                      prometheus.Gauge
	mqttBrokerPacketsReceived                    prometheus.Gauge
	mqttBrokerPacketsSent                        prometheus.Gauge
	mqttBrokerMemoryAlloc                        prometheus.Gauge
	mqttBrokerThreads                            prometheus.Gauge
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
	mqttBrokerTime = registerNewMQTTBrokerGauge(registry, "time", "Current time on the server.")
	mqttBrokerUptime = registerNewMQTTBrokerGauge(registry, "uptime", "The number of seconds the server has been online.")
	mqttBrokerBytesReceived = registerNewMQTTBrokerGauge(registry, "bytes_received", "Total number of bytes received since the broker started.")
	mqttBrokerBytesSent = registerNewMQTTBrokerGauge(registry, "bytes_sent", "Total number of bytes sent since the broker started.")
	mqttBrokerClientsConnected = registerNewMQTTBrokerGauge(registry, "clients_connected", "Number of currently connected clients.")
	mqttBrokerClientsDisconnected = registerNewMQTTBrokerGauge(registry, "clients_disconnected", "Total number of persistent clients (with clean session disabled) that are registered at the broker but are currently disconnected.")
	mqttBrokerClientsMaximum = registerNewMQTTBrokerGauge(registry, "clients_maximum", "Maximum number of active clients that have been connected.")
	mqttBrokerClientsTotal = registerNewMQTTBrokerGauge(registry, "clients_total", "Total number of connected and disconnected clients with a persistent session currently connected and registered.")
	mqttBrokerMessagesReceived = registerNewMQTTBrokerGauge(registry, "messages_received", "Total number of publish messages received.")
	mqttBrokerMessagesSent = registerNewMQTTBrokerGauge(registry, "messages_sent", "Total number of publish messages sent.")
	mqttBrokerMessagesDropped = registerNewMQTTBrokerGauge(registry, "messages_dropped", "Total number of publish messages dropped to slow subscriber.")
	mqttBrokerRetained = registerNewMQTTBrokerGauge(registry, "retained", "Total number of retained messages active on the broker.")
	mqttBrokerInflight = registerNewMQTTBrokerGauge(registry, "inflight", "The number of messages currently in-flight.")
	mqttBrokerInflightDropped = registerNewMQTTBrokerGauge(registry, "inflight_dropped", "The number of inflight messages which were dropped.")
	mqttBrokerSubscriptions = registerNewMQTTBrokerGauge(registry, "subscriptions", "Total number of subscriptions active on the broker.")
	mqttBrokerPacketsReceived = registerNewMQTTBrokerGauge(registry, "packets_received", "The total number of publish messages received.")
	mqttBrokerPacketsSent = registerNewMQTTBrokerGauge(registry, "packets_sent", "Total number of messages of any type sent since the broker started.")
	mqttBrokerMemoryAlloc = registerNewMQTTBrokerGauge(registry, "memory_alloc", "Memory currently allocated.")
	mqttBrokerThreads = registerNewMQTTBrokerGauge(registry, "threads", "Number of active goroutines, named as threads for platform ambiguity.")
	mqttBrokerSubscriptionManagerSubscribersSize = registerNewMQTTBrokerGauge(registry, "subscription_manager_subscribers_size", "The number of active subscribers in the subscription manager.")
	mqttBrokerSubscriptionManagerTopicsSize = registerNewMQTTBrokerGauge(registry, "subscription_manager_topics_size", "The number of active topics in the subscription manager.")
}

func collectMQTTBroker(server *mqtt.Server) {
	brokerSystemInfo := server.MQTTBroker.SystemInfo()

	mqttBrokerAppInfo.With(prometheus.Labels{
		"name":           Component.App().Info().Name,
		"version":        Component.App().Info().Version,
		"broker_version": brokerSystemInfo.Version,
	}).Set(1)
	mqttBrokerStarted.Set(float64(brokerSystemInfo.Started))
	mqttBrokerTime.Set(float64(brokerSystemInfo.Time))
	mqttBrokerUptime.Set(float64(brokerSystemInfo.Uptime))
	mqttBrokerBytesReceived.Set(float64(brokerSystemInfo.BytesReceived))
	mqttBrokerBytesSent.Set(float64(brokerSystemInfo.BytesSent))
	mqttBrokerClientsConnected.Set(float64(brokerSystemInfo.ClientsConnected))
	mqttBrokerClientsDisconnected.Set(float64(brokerSystemInfo.ClientsDisconnected))
	mqttBrokerClientsMaximum.Set(float64(brokerSystemInfo.ClientsMaximum))
	mqttBrokerClientsTotal.Set(float64(brokerSystemInfo.ClientsTotal))
	mqttBrokerMessagesReceived.Set(float64(brokerSystemInfo.MessagesReceived))
	mqttBrokerMessagesSent.Set(float64(brokerSystemInfo.MessagesSent))
	mqttBrokerMessagesDropped.Set(float64(brokerSystemInfo.MessagesDropped))
	mqttBrokerRetained.Set(float64(brokerSystemInfo.Retained))
	mqttBrokerInflight.Set(float64(brokerSystemInfo.Inflight))
	mqttBrokerInflightDropped.Set(float64(brokerSystemInfo.InflightDropped))
	mqttBrokerSubscriptions.Set(float64(brokerSystemInfo.Subscriptions))
	mqttBrokerPacketsReceived.Set(float64(brokerSystemInfo.PacketsReceived))
	mqttBrokerPacketsSent.Set(float64(brokerSystemInfo.PacketsSent))
	mqttBrokerMemoryAlloc.Set(float64(brokerSystemInfo.MemoryAlloc))
	mqttBrokerThreads.Set(float64(brokerSystemInfo.Threads))
	mqttBrokerSubscriptionManagerSubscribersSize.Set(float64(server.MQTTBroker.SubscribersSize()))
	mqttBrokerSubscriptionManagerTopicsSize.Set(float64(server.MQTTBroker.TopicsSize()))
}
