package mqtt

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/gohornet/inx-mqtt/pkg/mqtt"

	"github.com/iotaledger/hive.go/configuration"
	inx "github.com/iotaledger/inx/go"
)

var (
	// AppName name of the app.
	AppName = "inx-mqtt"
	// Version of the app.
	Version = "0.5.0"
)

const (
	APIRoute = "mqtt/v1"
)

func main() {
	fmt.Printf(">>>>> Starting %s v%s <<<<<\n", AppName, Version)

	config, err := loadConfigFile("config.json")
	if err != nil {
		panic(err)
	}

	conn, err := grpc.Dial(config.String(CfgINXAddress),
		grpc.WithChainUnaryInterceptor(grpc_retry.UnaryClientInterceptor(), grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := inx.NewINXClient(conn)
	server, err := NewServer(client,
		mqtt.WithBufferSize(config.Int(CfgMQTTBufferSize)),
		mqtt.WithBufferBlockSize(config.Int(CfgMQTTBufferBlockSize)),
		mqtt.WithTopicCleanupThreshold(config.Int(CfgMQTTTopicCleanupThreshold)),
		mqtt.WithWebsocketEnabled(config.Bool(CfgMQTTWebsocketEnabled)),
		mqtt.WithWebsocketBindAddress(config.String(CfgMQTTWebsocketBindAddress)),
		mqtt.WithTCPEnabled(config.Bool(CfgMQTTTCPEnabled)),
		mqtt.WithTCPBindAddress(config.String(CfgMQTTTCPBindAddress)),
		mqtt.WithTCPAuthEnabled(config.Bool(CfgMQTTTCPAuthEnabled)),
		mqtt.WithTCPAuthPasswordSalt(config.String(CfgMQTTTCPAuthPasswordSalt)),
		mqtt.WithTCPAuthUsers(config.StringMap(CfgMQTTTCPAuthUsers)),
		mqtt.WithTCPTLSEnabled(config.Bool(CfgMQTTTCPTLSEnabled)),
		mqtt.WithTCPTLSCertificatePath(config.String(CfgMQTTTCPTLSCertificatePath)),
		mqtt.WithTCPTLSPrivateKeyPath(config.String(CfgMQTTTCPTLSPrivateKeyPath)),
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		fmt.Println("Starting MQTT broker...")
		if err := server.Start(ctx); err != nil {
			panic(err)
		}
	}()

	if config.Bool(CfgPrometheusEnabled) {
		setupPrometheus(
			config.String(CfgPrometheusBindAddress),
			server,
			config.Bool(CfgPrometheusGoMetrics),
			config.Bool(CfgPrometheusProcessMetrics),
		)
	}

	var apiReq *inx.APIRouteRequest
	if config.Bool(CfgMQTTWebsocketEnabled) {
		bindAddressParts := strings.Split(config.String(CfgMQTTWebsocketBindAddress), ":")
		if len(bindAddressParts) != 2 {
			panic(fmt.Sprintf("invalid %s", CfgMQTTWebsocketBindAddress))
		}

		port, err := strconv.Atoi(bindAddressParts[1])
		if err != nil {
			panic(fmt.Sprintf("invalid %s", CfgMQTTWebsocketBindAddress))
		}

		apiReq = &inx.APIRouteRequest{
			Route: APIRoute,
			Host:  bindAddressParts[0],
			Port:  uint32(port),
		}

		fmt.Println("Registering API route...")
		if _, err := client.RegisterAPIRoute(context.Background(), apiReq); err != nil {
			panic(fmt.Errorf("failed to register API route via INX: %w", err))
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		select {
		case <-signalChan:
			done <- true
		case <-ctx.Done():
			done <- true
		}
	}()
	<-done
	cancel()

	// shutdown the broker
	server.Close()

	if apiReq != nil {
		fmt.Println("Removing API route...")
		if _, err := client.UnregisterAPIRoute(context.Background(), apiReq); err != nil {
			panic(fmt.Errorf("failed to remove API route via INX: %w", err))
		}
	}

	fmt.Println("exiting")
}

func retryBackoff(_ uint) time.Duration {
	return 2 * time.Second
}

func loadConfigFile(filePath string) (*configuration.Configuration, error) {
	config := configuration.New()
	if err := config.LoadFile(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading config file failed: %w", err)
	}

	fs := flagSet()
	flag.CommandLine.AddFlagSet(fs)
	flag.Parse()

	if err := config.LoadFlagSet(fs); err != nil {
		return nil, err
	}
	return config, nil
}
