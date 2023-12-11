package mqtt

import (
	"context"

	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/app"
	"github.com/iotaledger/hive.go/app/shutdown"
	"github.com/iotaledger/hive.go/ierrors"
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
	NodeBridge nodebridge.NodeBridge
	Server     *mqtt.Server
}

var (
	Component *app.Component
	deps      dependencies
)

func provide(c *dig.Container) error {

	type inDeps struct {
		dig.In
		NodeBridge nodebridge.NodeBridge
		*shutdown.ShutdownHandler
	}

	return c.Provide(func(deps inDeps) (*mqtt.Server, error) {
		broker, err := broker.NewBroker(
			broker.WithWebsocketEnabled(ParamsMQTT.Websocket.Enabled),
			broker.WithWebsocketBindAddress(ParamsMQTT.Websocket.BindAddress),
			broker.WithTCPEnabled(ParamsMQTT.TCP.Enabled),
			broker.WithTCPBindAddress(ParamsMQTT.TCP.BindAddress),
			broker.WithTCPTLSEnabled(ParamsMQTT.TCP.TLS.Enabled),
			broker.WithTCPTLSCertificatePath(ParamsMQTT.TCP.TLS.CertificatePath),
			broker.WithTCPTLSPrivateKeyPath(ParamsMQTT.TCP.TLS.PrivateKeyPath),
			broker.WithAuthPasswordSalt(ParamsMQTT.Auth.PasswordSalt),
			broker.WithAuthUsers(ParamsMQTT.Auth.Users),
			broker.WithPublicTopics(ParamsMQTT.PublicTopics),
			broker.WithProtectedTopics(ParamsMQTT.ProtectedTopics),
			broker.WithMaxTopicSubscriptionsPerClient(ParamsMQTT.Subscriptions.MaxTopicSubscriptionsPerClient),
			broker.WithTopicCleanupThresholdCount(ParamsMQTT.Subscriptions.TopicsCleanupThresholdCount),
			broker.WithTopicCleanupThresholdRatio(ParamsMQTT.Subscriptions.TopicsCleanupThresholdRatio),
			broker.WithMaximumClientWritesPending(ParamsMQTT.MaximumClientWritesPending),
			broker.WithClientWriteBufferSize(ParamsMQTT.ClientWriteBufferSize),
			broker.WithClientReadBufferSize(ParamsMQTT.ClientReadBufferSize),
		)
		if err != nil {
			return nil, ierrors.Wrap(err, "failed to create MQTT broker")
		}

		return mqtt.NewServer(
			Component.Logger(),
			deps.NodeBridge,
			broker,
			deps.ShutdownHandler,
			mqtt.WithWebsocketEnabled(ParamsMQTT.Websocket.Enabled),
			mqtt.WithWebsocketBindAddress(ParamsMQTT.Websocket.BindAddress),
			mqtt.WithWebsocketAdvertiseAddress(ParamsMQTT.Websocket.AdvertiseAddress),
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
