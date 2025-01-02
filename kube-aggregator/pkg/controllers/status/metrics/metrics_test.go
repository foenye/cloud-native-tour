package metrics

import (
	"k8s.io/component-base/metrics/testutil"
	"strings"
	"testing"
)

func TestAPIServiceAvailabilityCollection(t *testing.T) {
	collector := newAvailableCollector()

	availableAPIService := "available"
	unavailableAPIService := "unavailable"

	collector.SetAPIServiceAvailable(availableAPIService)
	collector.SetAPIServiceUnavailable(unavailableAPIService)

	if err := testutil.CustomCollectAndCompare(collector, strings.NewReader(`
	# HELP aggregator_unavailable_apiservice [ALPHA] Gauge of APIServices which are marked as unavailable broken down by APIService name.
	# TYPE aggregator_unavailable_apiservice gauge
	aggregator_unavailable_apiservice{name="available"} 0
	aggregator_unavailable_apiservice{name="unavailable"} 1
	`)); err != nil {
		t.Fatal(err)
	}

	collector.ClearState()

	collector.ForgetAPIService(availableAPIService)
	collector.ForgetAPIService(unavailableAPIService)

	if err := testutil.CustomCollectAndCompare(collector, strings.NewReader(`
	# HELP aggregator_unavailable_apiservice [ALPHA] Gauge of APIServices which are marked as unavailable broken down by APIService name.
	# TYPE aggregator_unavailable_apiservice gauge
	`)); err != nil {
		t.Fatal(err)
	}
}
