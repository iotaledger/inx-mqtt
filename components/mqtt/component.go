package mqtt

import (
	"context"

	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/shutdown"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	"github.com/iotaledger/inx-mqtt/pkg/daemon"
	"github.com/iotaledger/inx-mqtt/pkg/mqtt"
)

const (
	APIRoute = "mqtt/v2"
)

func init() {
	Component = &app.Component{
		Name:     "MQTT",
		DepsFunc: func(cDeps dependencies) { deps = cDeps },
		Params:   params,
		Provide:  provide,
		Run:      run,
	}
}

type dependencies struct {
	dig.In
	NodeBridge *nodebridge.NodeBridge
	Server     *Server
}

var (
	Component *app.Component
	deps      dependencies
)

func provide(c *dig.Container) error {

	type inDeps struct {
		dig.In
		NodeBridge *nodebridge.NodeBridge
		*shutdown.ShutdownHandler
	}

	return c.Provide(func(deps inDeps) (*Server, error) {
		return NewServer(
			Component.Logger(),
			deps.NodeBridge,
			deps.ShutdownHandler,
			mqtt.WithBufferSize(ParamsMQTT.BufferSize),
			mqtt.WithBufferBlockSize(ParamsMQTT.BufferBlockSize),
			mqtt.WithMaxTopicSubscriptionsPerClient(ParamsMQTT.Subscriptions.MaxTopicSubscriptionsPerClient),
			mqtt.WithTopicCleanupThresholdCount(ParamsMQTT.Subscriptions.TopicsCleanupThresholdCount),
			mqtt.WithTopicCleanupThresholdRatio(ParamsMQTT.Subscriptions.TopicsCleanupThresholdRatio),
			mqtt.WithWebsocketEnabled(ParamsMQTT.Websocket.Enabled),
			mqtt.WithWebsocketBindAddress(ParamsMQTT.Websocket.BindAddress),
			mqtt.WithWebsocketAdvertiseAddress(ParamsMQTT.Websocket.AdvertiseAddress),
			mqtt.WithTCPEnabled(ParamsMQTT.TCP.Enabled),
			mqtt.WithTCPBindAddress(ParamsMQTT.TCP.BindAddress),
			mqtt.WithTCPAuthEnabled(ParamsMQTT.TCP.Auth.Enabled),
			mqtt.WithTCPAuthPasswordSalt(ParamsMQTT.TCP.Auth.PasswordSalt),
			mqtt.WithTCPAuthUsers(ParamsMQTT.TCP.Auth.Users),
			mqtt.WithTCPTLSEnabled(ParamsMQTT.TCP.TLS.Enabled),
			mqtt.WithTCPTLSCertificatePath(ParamsMQTT.TCP.TLS.CertificatePath),
			mqtt.WithTCPTLSPrivateKeyPath(ParamsMQTT.TCP.TLS.PrivateKeyPath),
		)
	})
}

func run() error {
	return Component.Daemon().BackgroundWorker("MQTT", func(ctx context.Context) {
		Component.LogInfo("Starting MQTT Broker ...")
		deps.Server.Run(ctx)
		Component.LogInfo("Stopped MQTT Broker")
	}, daemon.PriorityStopMQTT)
}
