package metrics

import (
	apiregistrationv1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1helper "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1/helper"
	"k8s.io/component-base/metrics"
	"sync"
)

var (
	unavailableGaugeDesc = metrics.NewDesc(
		/* fqName */ "aggregator_unavailable_apiservice",
		/* help */ "Gauge of APIServices which are marked as unavailable broken down by APIService name.",
		/* variableLabels */ []string{"name"},
		/* constLabels */ nil,
		/* stabilityLevel */ metrics.ALPHA,
		/* deprecatedVersion */ "",
	)
)

type availabilityCollectorImplementation interface {
	// DescribeWithStability implements the metrics.StableCollector interface.
	DescribeWithStability(describeCh chan<- *metrics.Desc)
	// CollectWithStability implements the metrics.StableCollector interface.
	CollectWithStability(metricCh chan<- metrics.Metric)

	// setAPIServiceAvailability sets the given api service availability gauge to available or unavailable.
	setAPIServiceAvailability(apiServiceKey string, availability bool)
	// SetAPIServiceAvailable sets given api service availability gauge to available.
	SetAPIServiceAvailable(apiServiceKey string)
	// SetAPIServiceUnavailable sets given api service availability gauge to unavailable.
	SetAPIServiceUnavailable(apiServiceKey string)
	// ForgetAPIService removes the availability gauge of the given api service
	ForgetAPIService(apiServiceKey string)
}

// Check if apiServiceStatusCollector implements necessary interface.
var _ metrics.StableCollector = &availabilityCollector{}

var _ availabilityCollectorImplementation = &availabilityCollector{}

type availabilityCollector struct {
	metrics.BaseStableCollector

	mutex          sync.RWMutex
	availabilities map[string]bool
}

func newAvailableCollector() *availabilityCollector {
	return &availabilityCollector{availabilities: make(map[string]bool)}
}

// DescribeWithStability implements the metrics.StableCollector interface.
// Overrides from metrics.BaseStableCollector.
func (collector *availabilityCollector) DescribeWithStability(describeCh chan<- *metrics.Desc) {
	describeCh <- unavailableGaugeDesc
}

// CollectWithStability implements the metrics.StableCollector interface.
// Overrides from metrics.BaseStableCollector.
func (collector *availabilityCollector) CollectWithStability(metricCh chan<- metrics.Metric) {
	collector.mutex.RLock()
	defer collector.mutex.RUnlock()

	for apiServiceName, available := range collector.availabilities {
		gaugeValue := 1.0
		if available {
			gaugeValue = 0.0
		}
		metricCh <- metrics.NewLazyConstMetric(
			unavailableGaugeDesc,
			metrics.GaugeValue,
			gaugeValue,
			apiServiceName,
		)
	}
}

// setAPIServiceAvailability sets the given api service availability gauge to available or unavailable.
func (collector *availabilityCollector) setAPIServiceAvailability(apiServiceKey string, availability bool) {
	collector.mutex.Lock()
	defer collector.mutex.Unlock()

	collector.availabilities[apiServiceKey] = availability
}

// SetAPIServiceAvailable sets the given api service availability gauge to available.
func (collector *availabilityCollector) SetAPIServiceAvailable(apiServiceKey string) {
	collector.setAPIServiceAvailability(apiServiceKey, true)
}

// SetAPIServiceUnavailable sets given api service availability gauge to unavailable.
func (collector *availabilityCollector) SetAPIServiceUnavailable(apiServiceKey string) {
	collector.setAPIServiceAvailability(apiServiceKey, false)
}

// ForgetAPIService removes the availability gauge of the given api service
func (collector *availabilityCollector) ForgetAPIService(apiServiceKey string) {
	collector.mutex.Lock()
	defer collector.mutex.Unlock()

	delete(collector.availabilities, apiServiceKey)
}

type metricsImplementation interface {
	// Register registers API Service availability metrics.
	Register(
		registrationFunc func(metrics.Registerable) error,
		customRegistrationFunc func(metrics.StableCollector) error,
	) error
	// UnavailableCounter returns a counter to track API Service marked as unavailable.
	UnavailableCounter(apiServiceName, reason string) metrics.CounterMetric
	// SetUnavailableCounter increases the metrics only if the given service is unavailable and its
	// APIServiceCondition has changed
	SetUnavailableCounter(originalAPIService, newAPIService *apiregistrationv1.APIService)
	// SetUnavailableGauge set the metrics so that it reflect the current state base on availability
	// of the given service
	SetUnavailableGauge(newAPIService *apiregistrationv1.APIService)
}

var _ metricsImplementation = &Metrics{}

type Metrics struct {
	unavailableCounter *metrics.CounterVec
	*availabilityCollector
}

func New() *Metrics {
	return &Metrics{
		unavailableCounter: metrics.NewCounterVec(&metrics.CounterOpts{
			Name:           "aggregator_unavailable_apiservice_total",
			Help:           "Counter of APIServices which are marked as unavailable broken down by APIService name and reason.",
			StabilityLevel: metrics.ALPHA,
		}, []string{"name", "reason"}),
		availabilityCollector: newAvailableCollector(),
	}
}

// Register registers API service availability metrics.
func (metrics *Metrics) Register(registrationFunc func(metrics.Registerable) error, customRegistrationFunc func(metrics.StableCollector) error) error {
	if err := registrationFunc(metrics.unavailableCounter); err != nil {
		return err
	}

	if err := customRegistrationFunc(metrics.availabilityCollector); err != nil {
		return err
	}

	return nil
}

// UnavailableCounter returns a counter to track api services marked as unavailable.
func (metrics *Metrics) UnavailableCounter(apiServiceName, reason string) metrics.CounterMetric {
	return metrics.unavailableCounter.WithLabelValues(apiServiceName, reason)
}

// SetUnavailableCounter increases the metrics only if the given service is unavailable and its APIServiceCondition
// has changed.
func (metrics *Metrics) SetUnavailableCounter(originalAPIService, newAPIService *apiregistrationv1.APIService) {
	wasAvailable := apiregistrationv1helper.IsAPIServiceConditionTrue(originalAPIService, apiregistrationv1.Available)
	isAvailable := apiregistrationv1helper.IsAPIServiceConditionTrue(newAPIService, apiregistrationv1.Available)
	statusChanged := isAvailable != wasAvailable
	if statusChanged && !isAvailable {
		reason := "UnknownReason"
		if newCondition := apiregistrationv1helper.GetAPIServiceConditionByType(newAPIService, apiregistrationv1.Available); newCondition != nil {
			metrics.UnavailableCounter(newAPIService.Name, reason)
		}
	}
}

// SetUnavailableGauge sets the metrics so that it reflect the current state base on availability of the given service.
func (metrics *Metrics) SetUnavailableGauge(newAPIService *apiregistrationv1.APIService) {
	if apiregistrationv1helper.IsAPIServiceConditionTrue(newAPIService, apiregistrationv1.Available) {
		metrics.SetAPIServiceAvailable(newAPIService.Name)
		return
	}
	metrics.SetAPIServiceUnavailable(newAPIService.Name)
}
