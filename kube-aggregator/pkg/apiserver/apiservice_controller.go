package apiserver

import (
	"context"
	"fmt"
	apiregistrationv1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	generatedClientInformersV1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/informers/externalversions/apiregistration/v1"
	generatedClientListersV1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/listers/apiregistration/v1"
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/controllers"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"
)

// APIHandlerManager defines the behaviour that an API handler should have.
type APIHandlerManager interface {
	AddAPIService(apiService *apiregistrationv1.APIService) error
	RemoveAPIService(apiServiceName string)
}

type APIServiceRegistrationControllerImplementation interface {
	Run(stopCh <-chan struct{}, handlerSyncedCh chan<- struct{})
	dynamiccertificates.Listener

	sync(key string) error
	runWorker()
	processNextWorkItem() bool
	enqueueInternal(obj *apiregistrationv1.APIService)
	addAPIService(obj interface{})
	updateAPIService(oldObj, newObj interface{})
	deleteAPIService(obj interface{})
}

var _ APIServiceRegistrationControllerImplementation = &APIServiceRegistrationController{}

// APIServiceRegistrationController is responsible for registering and removing API services.
type APIServiceRegistrationController struct {
	apiHandlerManager APIHandlerManager

	apiServiceLister generatedClientListersV1.APIServiceLister
	apiServiceSynced cache.InformerSynced

	// To allow injection for testing.
	syncFn func(key string) error

	queue workqueue.TypedRateLimitingInterface[string]
}

// NewAPIServiceRegistrationController returns a new APIServiceRegistrationController.
func NewAPIServiceRegistrationController(apiServiceInformer generatedClientInformersV1.APIServiceInformer,
	apiHandlerManager APIHandlerManager) *APIServiceRegistrationController {
	c := &APIServiceRegistrationController{
		apiHandlerManager: apiHandlerManager,
		apiServiceLister:  apiServiceInformer.Lister(),
		apiServiceSynced:  apiServiceInformer.Informer().HasSynced,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "APIServiceRegistrationController"},
		),
	}

	_, _ = apiServiceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addAPIService,
		UpdateFunc: c.updateAPIService,
		DeleteFunc: c.deleteAPIService,
	})

	c.syncFn = c.sync

	return c
}

// Run starts APIServiceRegistrationController which will process all registration requests until stopCh is closed.
func (controller *APIServiceRegistrationController) Run(stopCh <-chan struct{}, handlerSyncedCh chan<- struct{}) {
	defer utilruntime.HandleCrash()
	defer controller.queue.ShutDown()

	klog.Info("Starting APIServiceRegistrationController")
	defer klog.Info("Shutting down APIServiceRegistrationController")

	if !controllers.WaitForCacheSync("APIServiceRegistrationController", stopCh, controller.apiServiceSynced) {
		return
	}

	/// initially sync all APIServices to make sure the proxy handler is complete
	if err := wait.PollUntilContextCancel(wait.ContextForChannel(stopCh), time.Minute, true, func(ctx context.Context) (done bool, err error) {
		services, err := controller.apiServiceLister.List(labels.Everything())
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to initially list APIServices: %v", err))
			return false, nil
		}
		for _, s := range services {
			if err := controller.apiHandlerManager.AddAPIService(s); err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to initially sync APIService %s: %v", s.Name, err))
				return false, nil
			}
		}
		return true, nil
	}); wait.Interrupted(err) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for proxy handler to initialize"))
		return
	} else if err != nil {
		panic(fmt.Errorf("unexpected error: %v", err))
	}
	close(handlerSyncedCh)

	// only start one worker thread since its a slow moving API and the aggregation server adding bits
	// aren't thread-safe
	go wait.Until(controller.runWorker, time.Second, stopCh)

	<-stopCh
}

// Enqueue queues all api services to be re-handled.
// This method is used by the controller to notify when the proxy cert content changes.
func (controller *APIServiceRegistrationController) Enqueue() {
	apiServices, err := controller.apiServiceLister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	for _, apiService := range apiServices {
		controller.addAPIService(apiService)
	}
}

func (controller *APIServiceRegistrationController) sync(key string) error {
	apiService, err := controller.apiServiceLister.Get(key)
	if errors.IsNotFound(err) {
		controller.apiHandlerManager.RemoveAPIService(key)
		return nil
	}
	if err != nil {
		return err
	}

	return controller.apiHandlerManager.AddAPIService(apiService)
}

func (controller *APIServiceRegistrationController) runWorker() {
	for controller.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false when it's time to quit.
func (controller *APIServiceRegistrationController) processNextWorkItem() bool {
	key, quit := controller.queue.Get()
	if quit {
		return false
	}
	defer controller.queue.Done(key)

	err := controller.syncFn(key)
	if err == nil {
		controller.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("%v failed with : %v", key, err))
	controller.queue.AddRateLimited(key)

	return true
}

func (controller *APIServiceRegistrationController) enqueueInternal(obj *apiregistrationv1.APIService) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Errorf("Couldn't get key for object %#v: %v", obj, err)
		return
	}

	controller.queue.Add(key)
}

func (controller *APIServiceRegistrationController) addAPIService(obj interface{}) {
	castObj := obj.(*apiregistrationv1.APIService)
	klog.V(4).Infof("Adding %s", castObj.Name)
	controller.enqueueInternal(castObj)
}

func (controller *APIServiceRegistrationController) updateAPIService(oldObj, _ interface{}) {
	castObj := oldObj.(*apiregistrationv1.APIService)
	klog.V(4).Infof("Updating %s", castObj.Name)
	controller.enqueueInternal(castObj)
}

func (controller *APIServiceRegistrationController) deleteAPIService(obj interface{}) {
	castObj, ok := obj.(*apiregistrationv1.APIService)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		castObj, ok = tombstone.Obj.(*apiregistrationv1.APIService)
		if !ok {
			klog.Errorf("Tombstone contained object that is not expected %#v", obj)
			return
		}
	}
	klog.V(4).Infof("Deleting %q", castObj.Name)
	controller.enqueueInternal(castObj)
}
