package prometheus

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/dig"

	"github.com/gohornet/inx-mqtt/core/mqtt"
	"github.com/gohornet/inx-mqtt/pkg/daemon"
	"github.com/iotaledger/hive.go/app"
)

func init() {
	Plugin = &app.Plugin{
		Status: app.StatusDisabled,
		Component: &app.Component{
			Name:      "Prometheus",
			DepsFunc:  func(cDeps dependencies) { deps = cDeps },
			Params:    params,
			Provide:   provide,
			Configure: configure,
			Run:       run,
		},
	}
}

type dependencies struct {
	dig.In
	PrometheusEcho *echo.Echo `name:"prometheusEcho"`
	Server         *mqtt.Server
}

var (
	Plugin *app.Plugin
	deps   dependencies
)

func provide(c *dig.Container) error {

	type depsOut struct {
		dig.Out
		PrometheusEcho *echo.Echo `name:"prometheusEcho"`
	}

	return c.Provide(func() depsOut {
		e := echo.New()
		e.HideBanner = true
		e.Use(middleware.Recover())

		return depsOut{
			PrometheusEcho: e,
		}
	})
}

func configure() error {

	registry := prometheus.NewRegistry()
	registerMQTTMetrics(registry)

	deps.PrometheusEcho.GET("/metrics", func(c echo.Context) error {

		collectMQTTBroker(deps.Server)

		handler := promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{
				EnableOpenMetrics: true,
			},
		)
		handler.ServeHTTP(c.Response().Writer, c.Request())
		return nil
	})

	return nil
}

func run() error {
	return Plugin.Daemon().BackgroundWorker("Prometheus exporter", func(ctx context.Context) {
		Plugin.LogInfo("Starting Prometheus exporter ... done")

		go func() {
			Plugin.LogInfof("You can now access the Prometheus exporter using: http://%s/metrics", ParamsPrometheus.BindAddress)
			if err := deps.PrometheusEcho.Start(ParamsPrometheus.BindAddress); err != nil && !errors.Is(err, http.ErrServerClosed) {
				Plugin.LogWarnf("Stopped Prometheus exporter due to an error (%s)", err)
			}
		}()

		<-ctx.Done()
		Plugin.LogInfo("Stopping Prometheus exporter ...")

		shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := deps.PrometheusEcho.Shutdown(shutdownCtx)
		if err != nil {
			Plugin.LogWarn(err)
		}
		shutdownCtxCancel()
		Plugin.LogInfo("Stopping Prometheus exporter ... done")
	}, daemon.PriorityStopPrometheus)
}
