package mqtt

import (
	"context"

	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/shutdown"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	"github.com/iotaledger/inx-mqtt/pkg/broker"
	"github.com/iotaledger/inx-mqtt/pkg/daemon"
	"github.com/iotaledger/inx-mqtt/pkg/mqtt"
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
	Server     *mqtt.Server
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

	return c.Provide(func(deps inDeps) (*mqtt.Server, error) {
		return mqtt.NewServer(
			Component.Logger(),
			deps.NodeBridge,
			deps.ShutdownHandler,
			broker.WithBufferSize(ParamsMQTT.BufferSize),
			broker.WithBufferBlockSize(ParamsMQTT.BufferBlockSize),
			broker.WithMaxTopicSubscriptionsPerClient(ParamsMQTT.Subscriptions.MaxTopicSubscriptionsPerClient),
			broker.WithTopicCleanupThresholdCount(ParamsMQTT.Subscriptions.TopicsCleanupThresholdCount),
			broker.WithTopicCleanupThresholdRatio(ParamsMQTT.Subscriptions.TopicsCleanupThresholdRatio),
			broker.WithWebsocketEnabled(ParamsMQTT.Websocket.Enabled),
			broker.WithWebsocketBindAddress(ParamsMQTT.Websocket.BindAddress),
			broker.WithWebsocketAdvertiseAddress(ParamsMQTT.Websocket.AdvertiseAddress),
			broker.WithTCPEnabled(ParamsMQTT.TCP.Enabled),
			broker.WithTCPBindAddress(ParamsMQTT.TCP.BindAddress),
			broker.WithTCPAuthEnabled(ParamsMQTT.TCP.Auth.Enabled),
			broker.WithTCPAuthPasswordSalt(ParamsMQTT.TCP.Auth.PasswordSalt),
			broker.WithTCPAuthUsers(ParamsMQTT.TCP.Auth.Users),
			broker.WithTCPTLSEnabled(ParamsMQTT.TCP.TLS.Enabled),
			broker.WithTCPTLSCertificatePath(ParamsMQTT.TCP.TLS.CertificatePath),
			broker.WithTCPTLSPrivateKeyPath(ParamsMQTT.TCP.TLS.PrivateKeyPath),
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
