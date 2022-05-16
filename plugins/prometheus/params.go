package prometheus

import (
	"github.com/iotaledger/hive.go/app"
)

type ParametersPrometheus struct {
	BindAddress    string `default:"localhost:9312" usage:"bind address on which the Prometheus HTTP server listens""`
	GoMetrics      bool   `default:"false" usage:"whether to include go metrics""`
	ProcessMetrics bool   `default:"false" usage:"whether to include process metrics""`
}

var ParamsPrometheus = &ParametersPrometheus{}

var params = &app.ComponentParams{
	Params: map[string]any{
		"prometheus": ParamsPrometheus,
	},
	Masked: nil,
}
