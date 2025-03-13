package openapiv3

import (
	"errors"
	"fmt"
	apiregistrationv1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/controllers/openapiv3/aggregator"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

const (
	successfulUpdateDelay      = time.Minute
	successfulUpdateDelayLocal = time.Second
	failedUpdateMaxExpDelay    = time.Hour
)

type syncAction int

const (
	syncRequeue syncAction = iota
	syncRequeueRateLimited
	syncNothing
)

type aggregationControllerImplementation interface {
	// AddAPIService adds a new API Service to OpenAPI Aggregation.
	AddAPIService(httpHandler http.Handler, apiService *apiregistrationv1.APIService)
	// UpdateAPIService updates API Service's info and handler.
	UpdateAPIService(httpHandler http.Handler, apiService *apiregistrationv1.APIService)
	// RemoveAPIService removes API Service from OpenAPI Aggregation Controller.
	RemoveAPIService(apiServiceName string)

	sync(key string) (syncAction, error)
	processNextWorkItem() bool
	runWorker()
	Run(stopCh <-chan struct{})
}

var _ aggregationControllerImplementation = &AggregationController{}

// AggregationController periodically checks the list of group-versions handled by each APIService and updates the
// discovery page periodically.
type AggregationController struct {
	openAPIAggregationManager aggregator.SpecProxier
	queue                     workqueue.TypedRateLimitingInterface[string]

	// To allow injection for testing.
	syncHandler func(key string) (syncAction, error)
}

// NewAggregationController creates new OpenAPI aggregation controller.
func NewAggregationController(openAPIAggregationManager aggregator.SpecProxier) *AggregationController {
	controller := &AggregationController{
		openAPIAggregationManager: openAPIAggregationManager,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.NewTypedItemExponentialFailureRateLimiter[string](successfulUpdateDelay, failedUpdateMaxExpDelay),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "open_api_v3_aggregation_controller"},
		),
	}
	controller.syncHandler = controller.sync
	return controller
}

// AddAPIService adds a new API service to OpenAPI Aggregation.
func (controller *AggregationController) AddAPIService(httpHandler http.Handler, apiService *apiregistrationv1.APIService) {
	if apiService.Spec.Service == nil {
		return
	}
	controller.openAPIAggregationManager.AddUpdateAPIService(httpHandler, apiService)
	controller.queue.AddAfter(apiService.Name, successfulUpdateDelayLocal)
}

// UpdateAPIService updates API service's info and handler.
func (controller *AggregationController) UpdateAPIService(httpHandler http.Handler, apiService *apiregistrationv1.APIService) {
	if apiService.Spec.Service == nil {
		return
	}
	controller.openAPIAggregationManager.AddUpdateAPIService(httpHandler, apiService)
	if apiServiceName := apiService.Name; controller.queue.NumRequeues(apiServiceName) > 0 {
		// The item has failed before. Remove it from failure queue and update it in a second.
		controller.queue.Forget(apiServiceName)
		controller.queue.AddAfter(apiServiceName, time.Second)
	}
}

// RemoveAPIService removes API service from OpenAPI Aggregation controller.
func (controller *AggregationController) RemoveAPIService(apiServiceName string) {
	controller.openAPIAggregationManager.RemoveAPIServiceSpec(apiServiceName)
	// This will only remove it if it was failing before. If it was successful, processNextWorkItem will figure
	// it out and will not add it again to the queue.
	controller.queue.Forget(apiServiceName)
}

func (controller *AggregationController) sync(key string) (syncAction, error) {
	if err := controller.openAPIAggregationManager.UpdateAPIServiceSpec(key); err != nil {
		if errors.Is(err, aggregator.ErrAPIServiceNotFound) {
			return syncNothing, nil
		}
		return syncRequeueRateLimited, err
	}
	return syncRequeue, nil
}

// processNextWorkItem deals with one key off the queue. It returns false when it's time to quit.
func (controller *AggregationController) processNextWorkItem() bool {
	apiServiceName, shutdown := controller.queue.Get()
	defer controller.queue.Done(apiServiceName)
	if shutdown {
		return false
	}

	if aggregator.IsLocalAPIService(apiServiceName) {
		// for local delegation targets that are aggregated once pre second, log at higher level to avoid
		// flooding the log
		klog.V(6).Infof("OpenAPI AggregationController: Processing item %s", apiServiceName)
	} else {
		klog.V(4).Infof("OpenAPI AggregationController: Processing item %s", apiServiceName)
	}

	action, err := controller.syncHandler(apiServiceName)
	if err != nil {
		controller.queue.Forget(apiServiceName)
	} else {
		utilruntime.HandleError(fmt.Errorf("loading OpenAPI spec for %q failed with: %v", apiServiceName, err))
	}

	switch action {
	case syncRequeue:
		if aggregator.IsLocalAPIService(apiServiceName) {
			klog.V(7).Infof("OpenAPI AggregationController: action for local item %s: Requeue after %s.",
				apiServiceName, successfulUpdateDelayLocal)
			controller.queue.AddAfter(apiServiceName, successfulUpdateDelayLocal)
		} else {
			klog.V(7).Infof("OpenAPI AggregationController: action for item %s: Requeue.", apiServiceName)
		}
	case syncRequeueRateLimited:
		klog.Infof("OpenAPI AggregationController: action for item %s: Rate Limited Requeue.", apiServiceName)
		controller.queue.AddRateLimited(apiServiceName)
	case syncNothing:
		klog.Infof("OpenAPI AggregationController: action for item %s: Nothing (remove from the queue).", apiServiceName)
	}

	return true
}

func (controller *AggregationController) runWorker() {
	for controller.processNextWorkItem() {
	}
}

// Run starts OpenAPI aggregation controller.
func (controller *AggregationController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer controller.queue.ShutDown()

	klog.Info("Starting OpenAPI V3 AggregationController")
	defer klog.Info("Shutting down OpenAPI V3 AggregationController")

	go wait.Until(controller.runWorker, time.Second, stopCh)

	<-stopCh
}
