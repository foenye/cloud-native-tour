package apiserver

import (
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

var x509MissingSANCounter = metrics.NewCounter(&metrics.CounterOpts{
	Namespace: "apiserver",
	Subsystem: "kube_aggregator",
	Name:      "x509_missing_san_total",
	Help: `Counts the number of requests to servers missing SAN extension in their serving certificate 
		OR the number of connection failures due to the lack of x509 certificate SAN extension missing (either/or, 
		based on the runtime environment)`,
	StabilityLevel: metrics.ALPHA,
})
var x509InsecureSHA1Counter = metrics.NewCounter(&metrics.CounterOpts{
	Namespace: "apiserver",
	Subsystem: "kube_aggregator",
	Name:      "x509_insecure_sha1_total",
	Help: `Counts the number of requests to servers with insecure SHA1 signatures in their serving certificate 
		OR the number of connection failures due to the insecure SHA1 signatures (either/or, based on the runtime environment)`,
	StabilityLevel: metrics.ALPHA,
})

func init() {
	legacyregistry.MustRegister(x509MissingSANCounter)
	legacyregistry.MustRegister(x509InsecureSHA1Counter)
}
