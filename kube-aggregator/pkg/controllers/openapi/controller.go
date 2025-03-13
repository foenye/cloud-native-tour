package openapi

import (
	"errors"
	"fmt"
	apiregistrationv1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/controllers/openapi/aggregator"
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
	processNextWorkItem() bool
	runWorker()
	Run(stopCh <-chan struct{})
	sync(key string) (syncAction, error)
	AddAPIService(httpHandler http.Handler, apiService *apiregistrationv1.APIService)
	UpdateAPIService(httpHandler http.Handler, apiService *apiregistrationv1.APIService)
	RemoveAPIService(apiServiceName string)
}

var _ aggregationControllerImplementation = &AggregationController{}

// AggregationController periodically check for changes in OpenAPI specs of APIServices and update/remove
// them if necessary.
type AggregationController struct {
	openAPIAggregationManager aggregator.SpecAggregator
	queue                     workqueue.TypedRateLimitingInterface[string]
	downloader                *aggregator.Downloader

	// To allow injection for testing.
	syncHandler func(key string) (syncAction, error)
}

// NewAggregationController creates new OpenAPI aggregation controller.
func NewAggregationController(
	downloader *aggregator.Downloader,
	openAPIAggregationManager aggregator.SpecAggregator) *AggregationController {
	controller := &AggregationController{
		openAPIAggregationManager: openAPIAggregationManager,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.NewTypedItemExponentialFailureRateLimiter[string](successfulUpdateDelay, failedUpdateMaxExpDelay),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "open_api_aggregation_controller"},
		),
		downloader: downloader,
	}
	controller.syncHandler = controller.sync
	return controller
}

// processNextWorkItem deals with one key off the queue. It returns false when it's time to quit.
func (controller *AggregationController) processNextWorkItem() bool {
	key, shutdown := controller.queue.Get()
	defer controller.queue.Done(key)
	if shutdown {
		return false
	}
	klog.V(4).Infof("OpenAPI AggregationController: Process item %s", key)

	action, err := controller.syncHandler(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("locading OpenAPI spec for %q failed with: %v", key, err))
	}

	switch action {
	case syncRequeue:
		controller.queue.AddAfter(key, successfulUpdateDelay)
	case syncRequeueRateLimited:
		klog.Infof("OpenAPI AggregationController: action for item %s: Rate Limited Requeue.", key)
		controller.queue.AddRateLimited(key)
	case syncNothing:
		controller.queue.Forget(key)
	}
	return true
}

func (controller *AggregationController) runWorker() {
	for controller.processNextWorkItem() {
	}
}

// Run starts OpenAPI AggregationController
func (controller *AggregationController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer controller.queue.ShutDown()

	klog.Info("Starting OpenAPI AggregationController")
	defer klog.Info("Shutting down OpenAPI AggregationController")

	go wait.Until(controller.runWorker, time.Second, stopCh)
}

func (controller *AggregationController) sync(key string) (syncAction, error) {
	if err := controller.openAPIAggregationManager.UpdateAPIServiceSpec(key); err != nil {
		switch {
		case errors.Is(err, aggregator.ErrAPIServiceNotFound):
			return syncNothing, nil
		default:
			return syncRequeueRateLimited, nil
		}
	}
	return syncRequeue, nil
}

// AddAPIService adds a new API Service to OpenAPI Aggregation.
func (controller *AggregationController) AddAPIService(httpHandler http.Handler, apiService *apiregistrationv1.APIService) {
	if apiService.Spec.Service == nil {
		return
	}
	if err := controller.openAPIAggregationManager.AddUpdateAPIService(apiService, httpHandler); err != nil {
		utilruntime.HandleError(fmt.Errorf("adding %q to AggregationController failed with: %v", apiService.Name, err))
	}
	controller.queue.AddAfter(apiService.Name, successfulUpdateDelayLocal)
}

// UpdateAPIService updates API Service's info and handler.
func (controller *AggregationController) UpdateAPIService(_ http.Handler, apiService *apiregistrationv1.APIService) {
	if apiService.Spec.Service == nil {
		return
	}
	if err := controller.openAPIAggregationManager.UpdateAPIServiceSpec(apiService.Name); err != nil {
		utilruntime.HandleError(fmt.Errorf("error updating APIService %q with error: %v", apiService.Name, err))
	}

	if key := apiService.Name; controller.queue.NumRequeues(key) > 0 {
		// The item has failed before. Remove it from failure queue and update it in a second.
		controller.queue.Forget(key)
		controller.queue.AddAfter(key, successfulUpdateDelayLocal)
	}

	// Else: The item has been succeeded before and it will be updated soon (after successfulUpdateDelay)
	// we don't add it again as it will cause a duplication of items.
}

// RemoveAPIService removes API service from OpenAPI Aggregation Controller.
func (controller *AggregationController) RemoveAPIService(apiServiceName string) {
	controller.openAPIAggregationManager.RemoveAPIService(apiServiceName)
	// This will only remove it if it was failing before. If it was successful, processNextWorkItem will figure it out
	// and will not add it again to the queue.
	controller.queue.Forget(apiServiceName)
}
