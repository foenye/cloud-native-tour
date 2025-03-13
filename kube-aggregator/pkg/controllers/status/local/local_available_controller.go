package local

import (
	"context"
	"fmt"
	apiregistrationv1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1helper "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1/helper"
	generatedClientsetTypedV1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
	generatedClientInformersV1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/informers/externalversions/apiregistration/v1"
	generatedClientListersV1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/listers/apiregistration/v1"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/controllers"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/controllers/status/metrics"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"
)

type availableConditionControllerImplementation interface {
	processNextWorkItem() bool
	runWorker()
	Run(workers int, stopCh <-chan struct{})
	updateAPIServiceStatus(originalAPIService, newAPIService *apiregistrationv1.APIService) (*apiregistrationv1.APIService, error)
	sync(key string) error
	addAPIService(obj interface{})
	updateAPIService(oldObj, newObj interface{})
	deleteAPIService(obj interface{})
}

var _ availableConditionControllerImplementation = &AvailableConditionController{}

// AvailableConditionController handles checking the availability of registered local API services.
type AvailableConditionController struct {
	apiServiceClient generatedClientsetTypedV1.APIServicesGetter

	apiServiceLister generatedClientListersV1.APIServiceLister
	apiServiceSynced cache.InformerSynced

	// To allow injection for testing.
	syncFn func(key string) error

	queue workqueue.TypedRateLimitingInterface[string]

	// metrics registered into legacy registry
	metrics *metrics.Metrics
}

// New returns a new local availability AvailableConditionController.
func New(
	apiServiceInformer generatedClientInformersV1.APIServiceInformer,
	apiServiceClient generatedClientsetTypedV1.APIServicesGetter,
	metrics *metrics.Metrics,
) (*AvailableConditionController, error) {
	controller := &AvailableConditionController{
		apiServiceClient: apiServiceClient,
		apiServiceLister: apiServiceInformer.Lister(),
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			/**
			We want a fairly tight requeue time.  The controller listens to the API, but because it relies on one the
			rout-ability of the service network, it's possible for an external, non-watchable factor to affect
			availability.  This keeps the maximum disruption time to a minimum, but it does prevent hot loops.
			*/
			workqueue.NewTypedItemExponentialFailureRateLimiter[string](5*time.Millisecond, 30*time.Second),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "LocalAvailabilityController"},
		),
		metrics: metrics,
	}

	// re-sync on this one because it is low cardinality and rechecking the actual discovery allows us to detect
	// health in a more timely fashion when network connectivity to nodes is snipped, but the network still attempts
	// to route here.  See https://github.com/openshift/origin/issues/17159#issuecomment-341798063
	handlerWithReSyncPeriod, _ := apiServiceInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.addAPIService,
			UpdateFunc: controller.updateAPIService,
			DeleteFunc: controller.deleteAPIService,
		},
		30*time.Second,
	)
	controller.apiServiceSynced = handlerWithReSyncPeriod.HasSynced

	controller.syncFn = controller.sync

	return controller, nil
}

// processNextWorkItem deals with one key off the queue. It returns false when it is time to quited.
func (controller *AvailableConditionController) processNextWorkItem() bool {
	item, quited := controller.queue.Get()
	if quited {
		return false
	}
	defer controller.queue.Done(item)

	if err := controller.syncFn(item); err != nil {
		controller.queue.Forget(item)
	} else {
		utilruntime.HandleError(fmt.Errorf("%v failed with: %w", item, err))
		controller.queue.AddRateLimited(item)
	}
	return true
}

func (controller *AvailableConditionController) runWorker() {
	for controller.processNextWorkItem() {
	}
}

// Run starts the AvailableConditionController loop which manages the availability condition of API services.
func (controller *AvailableConditionController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer controller.queue.ShutDown()

	klog.Info("Starting LocalAvailability controller")
	defer klog.Info("Shutting down LocalAvailability controller")

	// This waits not just for the informers to sync, but for our handlers to be called; since the handlers are
	// three different ways of enqueueing the same thing, waiting for these permits the queue to maximally
	// de-duplicate the entries.
	if !controllers.WaitForCacheSync("LocalAvailability", stopCh, controller.apiServiceSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(controller.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

// updateAPIServiceStatus only issues an update if a change is detected. We have a tight reSync loop to quickly detect
// dead api services. Doing that means we don't want to quickly issue no-op updates.
func (controller *AvailableConditionController) updateAPIServiceStatus(originalAPIService, newAPIService *apiregistrationv1.APIService) (*apiregistrationv1.APIService, error) {
	// update this metric on every sync operation to reflect the actual state
	controller.metrics.SetUnavailableGauge(newAPIService)

	if equality.Semantic.DeepEqual(originalAPIService.Status, newAPIService.Status) {
		return newAPIService, nil
	}

	conditionRaw := apiregistrationv1helper.GetAPIServiceConditionByType(originalAPIService, apiregistrationv1.Available)
	conditionNow := apiregistrationv1helper.GetAPIServiceConditionByType(newAPIService, apiregistrationv1.Available)

	conditionUnknown := apiregistrationv1.APIServiceCondition{
		Type:   apiregistrationv1.Available,
		Status: apiregistrationv1.ConditionUnknown,
	}
	if conditionRaw == nil {
		conditionRaw = &conditionUnknown
	}
	if conditionNow == nil {
		conditionNow = &conditionUnknown
	}

	if *conditionRaw != *conditionNow {
		klog.V(2).InfoS("changing APIService availability",
			"name", newAPIService.Name,
			"oldStatus", conditionRaw.Status,
			"newStatus", conditionNow.Status,
			"message", conditionNow.Message,
			"reason", conditionNow.Reason,
		)
	}

	newAPIService, err := controller.apiServiceClient.APIServices().UpdateStatus(context.TODO(), newAPIService, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	controller.metrics.SetUnavailableCounter(originalAPIService, newAPIService)
	return newAPIService, nil
}

func (controller *AvailableConditionController) sync(key string) error {
	rawAPIService, err := controller.apiServiceLister.Get(key)
	if errors.IsNotFound(err) {
		controller.metrics.ForgetAPIService(key)
		return nil
	}
	if err != nil {
		return err
	}

	if rawAPIService.Spec.Service != nil {
		return nil
	}

	// local API services are always considered available
	apiService := rawAPIService.DeepCopy()
	apiregistrationv1helper.SetAPIServiceCondition(apiService, apiregistrationv1helper.NewLocalAvailableAPIServiceCondition())
	_, err = controller.updateAPIServiceStatus(rawAPIService, apiService)
	return err
}

func (controller *AvailableConditionController) addAPIService(obj interface{}) {
	apiService := obj.(*apiregistrationv1.APIService)
	klog.V(4).Infof("Adding %s", apiService.Name)
	controller.queue.Add(apiService.Name)
}

func (controller *AvailableConditionController) updateAPIService(oldObj,
	_ interface{}, // newObj
) {
	rawAPIService := oldObj.(*apiregistrationv1.APIService)
	klog.V(4).Infof("Updating %s", rawAPIService.Name)
	controller.queue.Add(rawAPIService.Name)
}

func (controller *AvailableConditionController) deleteAPIService(obj interface{}) {
	apiService, casted := obj.(*apiregistrationv1.APIService)
	if !casted {
		tombstone, casted := obj.(cache.DeletedFinalStateUnknown)
		if !casted {
			klog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		apiService, casted = tombstone.Obj.(*apiregistrationv1.APIService)
		if !casted {
			klog.Errorf("Tombstone contained object that is not expected %#v", obj)
			return
		}
	}
	klog.V(4).Infof("Deleting %s", apiService.Name)
	controller.queue.Add(apiService.Name)
}
