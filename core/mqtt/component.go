package mqtt

import (
	"context"

	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/core/app"
	"github.com/iotaledger/hive.go/core/app/pkg/shutdown"
	"github.com/iotaledger/inx-app/nodebridge"
	"github.com/iotaledger/inx-mqtt/pkg/daemon"
	"github.com/iotaledger/inx-mqtt/pkg/mqtt"
)

const (
	APIRoute = "mqtt/v1"
)

func init() {
	CoreComponent = &app.CoreComponent{
		Component: &app.Component{
			Name:     "MQTT",
			DepsFunc: func(cDeps dependencies) { deps = cDeps },
			Params:   params,
			Provide:  provide,
			Run:      run,
		},
	}
}

type dependencies struct {
	dig.In
	NodeBridge *nodebridge.NodeBridge
	Server     *Server
}

var (
	CoreComponent *app.CoreComponent
	deps          dependencies
)

func provide(c *dig.Container) error {

	type inDeps struct {
		dig.In
		NodeBridge *nodebridge.NodeBridge
		*shutdown.ShutdownHandler
	}

	return c.Provide(func(deps inDeps) (*Server, error) {
		return NewServer(
			CoreComponent.Logger(),
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
	return CoreComponent.Daemon().BackgroundWorker("MQTT", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting MQTT Broker ...")
		deps.Server.Run(ctx)
		CoreComponent.LogInfo("Stopped MQTT Broker")
	}, daemon.PriorityStopMQTT)
}
