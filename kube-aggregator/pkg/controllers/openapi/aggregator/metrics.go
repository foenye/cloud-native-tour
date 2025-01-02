package aggregator

import (
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

var (
	regenerationCounter = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Name: "aggregator_openapi_v2_regeneration_count",
			Help: "Counter of OpenAPI v2 spec regeneration count broken down by causing APIService name and " +
				"reason.",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"apiservice", "reason"},
	)
	regenerationDurationGauge = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Name:           "aggregator_openapi_v2_regeneration_duration",
			Help:           "Gauge of OpenAPI v2 spec regeneration duration in seconds.",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"reason"},
	)
)

func init() {
	legacyregistry.MustRegister(regenerationCounter)
	legacyregistry.MustRegister(regenerationDurationGauge)
}
