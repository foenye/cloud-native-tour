package autoregister

import (
	"context"
	"fmt"
	apiregistrationv1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	generatedClientsetTypedV1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
	generatedClientInformersV1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/informers/externalversions/apiregistration/v1"
	generatedClientListersV1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/listers/apiregistration/v1"
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/controllers"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"reflect"
	"sync"
	"time"
)

const (
	// ManagedLabel is a label attached to the APIService that identifies how the APIService wants to be synced.
	ManagedLabel = "kube-aggregator.kubernetes.io/automanaged"
	// manageOnStart is a value for the auto register ManagedLabel that indicates the APIService wants to be synced
	// one time when the controller starts.
	manageOnStart = "onstart"
	// manageContinuously is a value for the auto register ManagedLabel that indicates the APIService wants to be synced
	// continuously.
	manageContinuously = "true"
)

// AutoAPIServiceRegistration is an interface which callers can re-declare locally and properly cast to for adding and
// removing APIServices.
type AutoAPIServiceRegistration interface {
	// AddAPIServiceToSyncOnStart adds an API service to sync on start.
	AddAPIServiceToSyncOnStart(in *apiregistrationv1.APIService)
	// AddAPIServiceToSync adds an API service to sync continuously.
	AddAPIServiceToSync(in *apiregistrationv1.APIService)
	// RemoveAPIServiceToSync removes an API service to auto-register.
	RemoveAPIServiceToSync(name string)
}

// controller is auto register used to keep a particular set of APIServices present in the API. It is useful for cases
// where you want to auto-register APIs like TPRs or groups from the core kube-apiserver.
type controller struct {
	apiServiceLister generatedClientListersV1.APIServiceLister
	apiServiceSynced cache.InformerSynced
	apiServiceClient generatedClientsetTypedV1.APIServicesGetter

	apiServicesToSyncLock sync.RWMutex
	apiServicesToSync     map[string]*apiregistrationv1.APIService

	syncHandler func(apiServiceName string) error

	// track which services we hava synced
	syncedSuccessfullyLock *sync.RWMutex
	syncedSuccessfully     map[string]bool

	// remember names of services that existed when we started
	apiServicesAtStart map[string]bool

	// queue is where incoming work is placed to de-dup and to allow "easy" rate limited requeues on errors
	queue workqueue.TypedRateLimitingInterface[string]
}

type Controller struct {
	*controller
}

func NewAutoRegisterController(apiServiceInformer generatedClientInformersV1.APIServiceInformer,
	apiServiceClient generatedClientsetTypedV1.APIServicesGetter) *Controller {
	c := &controller{
		apiServiceLister:  apiServiceInformer.Lister(),
		apiServiceSynced:  apiServiceInformer.Informer().HasSynced,
		apiServiceClient:  apiServiceClient,
		apiServicesToSync: map[string]*apiregistrationv1.APIService{},

		apiServicesAtStart: map[string]bool{},

		syncedSuccessfullyLock: &sync.RWMutex{},
		syncedSuccessfully:     map[string]bool{},

		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "autoregister"},
		),
	}
	c.syncHandler = c.checkAPIService

	_, _ = apiServiceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			apiService := obj.(*apiregistrationv1.APIService)
			c.queue.Add(apiService.Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			apiService := newObj.(*apiregistrationv1.APIService)
			c.queue.Add(apiService.Name)
		},
		DeleteFunc: func(obj interface{}) {
			apiService, casted := obj.(*apiregistrationv1.APIService)
			if !casted {
				tombstone, casted := obj.(cache.DeletedFinalStateUnknown)
				if !casted {
					klog.V(2).Infof("Couldn't get object from tombstone %#v", obj)
					return
				}
				apiService, casted = tombstone.Obj.(*apiregistrationv1.APIService)
				if !casted {
					klog.V(2).Infof("Tombstone contained unexpected object %#v", obj)
					return
				}
			}
			c.queue.Add(apiService.Name)
		},
	})

	return &Controller{c}
}

// processNextWorkItem deals with one key off the queue. It returns false when it's time to quit.
func (c *controller) processNextWorkItem() bool {
	// pull the next work item from queue. It should be a key we use to lookup something in a cache
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// you always hava to indicate to the queue that you've completed a piece of work
	defer c.queue.Done(key)

	// do your work on the key. This method will contains your "do stuff" logic
	err := c.syncHandler(key)
	if err == nil {
		// if you had no error, tell the queue to stop tracking history for your key. This will rest things like
		// failure counts for pre-item rate limiting
		c.queue.Forget(key)
		return true
	}

	// there was a failure so be sure to report it. This method allows for pluggable error handling which can be
	// used for things like cluster-monitoring
	utilruntime.HandleError(fmt.Errorf("%v failed with %v", key, err))

	// since we failed, we should requeue the item to work on later. This method will add a backoff to avoid
	// hotlooping on particular items (they're probably still not going to work right away) and overall controller
	// protection (everything I've done is broken, this controller needs to calm down or it can starve other useful
	// work) cases.
	c.queue.AddRateLimited(key)

	return true
}

func (c *controller) runWorker() {
	// hot loop until we're told to stop. processNextWorkItem will automatically wait until there's work available,
	// so we don't worry about secondary waits
	for c.processNextWorkItem() {
	}
}

// Run starts the autoregister controller in a loop which syncs API services until stopCh is closed.
func (c *controller) Run(workers int, stopCh <-chan struct{}) {
	// don't let panics crash the process
	defer utilruntime.HandleCrash()
	// make sure the work queue is shutdown which will trigger workers to end
	defer c.queue.ShutDown()

	klog.Info("Starting auto register controller")
	defer klog.Infof("Shutting down auto register contoller")

	// wait for your secondary caches to fill before starting your work
	if !controllers.WaitForCacheSync("autoregister", stopCh, c.apiServiceSynced) {
		return
	}

	// record APIService objects that existed when we started
	if services, err := c.apiServiceLister.List(labels.Everything()); err != nil {
		for _, service := range services {
			c.apiServicesAtStart[service.Name] = true
		}
	}

	// start up your worker threads based on workers. Some controllers hava multiple kinds of workers
	for i := 0; i < workers; i++ {
		// runWorker will loop until "something bad" happens. The .Until will then rekick the worker after one second
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	// wait until we're told to stop
	<-stopCh
}

// checkAPIService syncs the current APIService against a list of desired APIService objects
//
//	                                                | A. desired: not found | B. desired: sync on start | C. desired: sync always
//	------------------------------------------------|-----------------------|---------------------------|------------------------
//	1. current: lookup error                        | error                 | error                     | error
//	2. current: not found                           | -                     | create once               | create
//	3. current: no sync                             | -                     | -                         | -
//	4. current: sync on start, not present at start | -                     | -                         | -
//	5. current: sync on start, present at start     | delete once           | update once               | update once
//	6. current: sync always                         | delete                | update once               | update
func (c *controller) checkAPIService(name string) (err error) {
	desired := c.GetAPIServiceToSync(name)
	curr, err := c.apiServiceLister.Get(name)

	// if we've never synced this service successfully, record a successful sync.
	hasSynced := c.hasSyncedSuccessfully(name)
	if !hasSynced {
		defer func() {
			if err == nil {
				c.setSyncedSuccessfully(name)
			}
		}()
	}

	switch {
	// we had a real error, just return it (1A,1B,1C)
	case err != nil && !apiErrors.IsNotFound(err):
		return err

	// we don't have an entry and we don't want one (2A)
	case apiErrors.IsNotFound(err) && desired == nil:
		return nil

	// the local object only wants to sync on start and has already synced (2B,5B,6B "once" enforcement)
	case isAutoManagedOnStart(desired) && hasSynced:
		return nil

	// we don't hava entry and we do want one (2B,2C)
	case apiErrors.IsNotFound(err) && desired != nil:
		_, err := c.apiServiceClient.APIServices().Create(context.TODO(), desired, metav1.CreateOptions{})
		if apiErrors.IsAlreadyExists(err) {
			return nil
		}
		return err

	// we aren't trying to manage this APIService (3A,3B,3C)
	case !isAutoManaged(curr):
		return nil

	// the remote object only wants to sync on start, but was added after we started (4A,4B,4C)
	case isAutoManagedOnStart(curr) && !c.apiServicesAtStart[name]:
		return nil

	// the remote object only wants to sync on start and has already synced (5A,5B,5C "once" enforcement)
	case isAutoManagedOnStart(curr) && hasSynced:
		return nil

	// we have a spurious APIService that we're managing, delete it (5A,6A)
	case desired == nil:
		deleteOptions := metav1.DeleteOptions{Preconditions: metav1.NewUIDPreconditions(string(curr.UID))}
		err := c.apiServiceClient.APIServices().Delete(context.TODO(), curr.Name, deleteOptions)
		if apiErrors.IsNotFound(err) || apiErrors.IsConflict(err) {
			return nil
		}
		return err

	// if the specs already match, nothing for us to do
	case reflect.DeepEqual(curr.Spec, desired.Spec):
		return nil
	}

	// we have an entry and we have desired, now we deconflict. Only a few fields matter. (5B,5C,6B,6C)
	apiService := curr.DeepCopy()
	apiService.Spec = desired.Spec
	_, err = c.apiServiceClient.APIServices().Update(context.TODO(), apiService, metav1.UpdateOptions{})
	if apiErrors.IsNotFound(err) || apiErrors.IsConflict(err) {
		return nil
	}
	return err
}

// GetAPIServiceToSync gets a single API service to sync.
func (c *controller) GetAPIServiceToSync(name string) *apiregistrationv1.APIService {
	c.apiServicesToSyncLock.RLock()
	defer c.apiServicesToSyncLock.RUnlock()

	return c.apiServicesToSync[name]
}

func (c *controller) hasSyncedSuccessfully(name string) bool {
	c.syncedSuccessfullyLock.RLock()
	defer c.syncedSuccessfullyLock.RUnlock()
	return c.syncedSuccessfully[name]
}

func (c *controller) setSyncedSuccessfully(name string) {
	c.syncedSuccessfullyLock.Lock()
	defer c.syncedSuccessfullyLock.Unlock()
	c.syncedSuccessfully[name] = true
}

// controller implements AutoAPIServiceRegistration interface.
// - AddAPIServiceToSyncOnStart
// - AddAPIServiceToSync
// - RemoveAPIServiceToSync
var _ AutoAPIServiceRegistration = &controller{}

// AddAPIServiceToSyncOnStart registers an API service to sync only when the controller starts.
func (c *controller) AddAPIServiceToSyncOnStart(in *apiregistrationv1.APIService) {
	c.addAPIServiceToSync(in, manageOnStart)
}

// AddAPIServiceToSync  registers an API service to sync continuously.
func (c *controller) AddAPIServiceToSync(in *apiregistrationv1.APIService) {
	c.addAPIServiceToSync(in, manageContinuously)
}

// RemoveAPIServiceToSync deletes a registered API service.
func (c *controller) RemoveAPIServiceToSync(name string) {
	c.apiServicesToSyncLock.Lock()
	defer c.apiServicesToSyncLock.Unlock()

	delete(c.apiServicesToSync, name)
	c.queue.Add(name)
}

func (c *controller) addAPIServiceToSync(in *apiregistrationv1.APIService, syncType string) {
	c.apiServicesToSyncLock.Lock()
	defer c.apiServicesToSyncLock.Unlock()

	apiService := in.DeepCopy()
	if apiService.Labels == nil {
		apiService.Labels = map[string]string{}
	}
	apiService.Labels[ManagedLabel] = syncType

	c.apiServicesToSync[apiService.Name] = apiService
	c.queue.Add(apiService.Name)
}

func isAutoManagedOnStart(apiService *apiregistrationv1.APIService) bool {
	return autoManagedType(apiService) == manageOnStart
}

func isAutoManaged(apiService *apiregistrationv1.APIService) bool {
	managedType := autoManagedType(apiService)
	return managedType == manageOnStart || managedType == manageContinuously
}

func autoManagedType(apiService *apiregistrationv1.APIService) string {
	if apiService == nil {
		return ""
	}
	return apiService.Labels[ManagedLabel]
}
