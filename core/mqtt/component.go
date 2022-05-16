package mqtt

import (
	"context"
	"fmt"

	"go.uber.org/dig"

	"github.com/gohornet/inx-mqtt/pkg/daemon"
	"github.com/gohornet/inx-mqtt/pkg/mqtt"
	"github.com/gohornet/inx-mqtt/pkg/nodebridge"
	"github.com/iotaledger/hive.go/app"
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
	}

	return c.Provide(func(deps inDeps) (*Server, error) {
		return NewServer(deps.NodeBridge,
			mqtt.WithBufferSize(ParamsMQTT.BufferSize),
			mqtt.WithBufferBlockSize(ParamsMQTT.BufferBlockSize),
			mqtt.WithTopicCleanupThreshold(ParamsMQTT.TopicCleanupThreshold),
			mqtt.WithWebsocketEnabled(ParamsMQTT.Websocket.Enabled),
			mqtt.WithWebsocketBindAddress(ParamsMQTT.Websocket.BindAddress),
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
		CoreComponent.LogInfo("Starting MQTT Broker")

		mqttCtx, cancel := context.WithCancel(ctx)
		go func() {
			fmt.Println("Starting MQTT broker...")
			if err := deps.Server.Start(mqttCtx); err != nil {
				panic(err)
			}
		}()

		if ParamsMQTT.Websocket.Enabled {
			CoreComponent.LogInfo("Registering API route...")
			if err := deps.NodeBridge.RegisterAPIRoute(APIRoute, ParamsMQTT.Websocket.BindAddress); err != nil {
				CoreComponent.LogWarnf("failed to register API route via INX: %w", err)
			}
		}

		<-ctx.Done()
		cancel()

		// shutdown the broker
		deps.Server.Close()

		if ParamsMQTT.Websocket.Enabled {
			CoreComponent.LogInfo("Removing API route...")
			if err := deps.NodeBridge.UnregisterAPIRoute(APIRoute); err != nil {
				CoreComponent.LogWarnf("failed to remove API route via INX: %w", err)
			}
		}
		CoreComponent.LogInfo("Stopped MQTT Broker")
	}, daemon.PriorityStopMQTT)
}
