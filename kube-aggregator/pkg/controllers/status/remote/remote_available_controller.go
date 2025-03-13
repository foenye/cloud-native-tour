package remote

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
	kubeAPICorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientGoInformersCoreV1 "k8s.io/client-go/informers/core/v1"
	clientGoListersCoreV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/transport"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"time"
)

type certKeyFn func() ([]byte, []byte)

// ServiceResolver knows how to convert a service reference into an actual location.
type ServiceResolver interface {
	ResolveEndpoint(namespace, name string, port int32) (*url.URL, error)
}

type availableConditionImplementation interface {
	processNextWorkItem() bool
	runWorker()
	Run(workers int, stopCh <-chan struct{})
	updateAPIServiceStatus(originalAPIService, newAPIService *apiregistrationv1.APIService) (*apiregistrationv1.APIService, error)
	sync(key string) error
	rebuildAPIServiceCache()

	addAPIService(obj interface{})
	updateAPIService(oldObj, newObj interface{})
	deleteAPIService(obj interface{})

	getAPIServicesFor(obj runtime.Object) []string
	addService(obj interface{})
	updateService(oldObj, newObj interface{})
	deleteService(obj interface{})
	addEndpoints(obj interface{})
	updateEndpoints(oldObj, newObj interface{})
	deleteEndpoints(obj interface{})
}

var _ availableConditionImplementation = &AvailableConditionController{}

// AvailableConditionController handles checking the availability of registered API services.
type AvailableConditionController struct {
	apiServiceClient generatedClientsetTypedV1.APIServicesGetter

	apiServiceLister generatedClientListersV1.APIServiceLister
	apiServiceSynced cache.InformerSynced

	// serviceLister is used to get the IP to create the transport for
	serviceLister  clientGoListersCoreV1.ServiceLister
	servicesSynced cache.InformerSynced

	endpointsLister clientGoListersCoreV1.EndpointsLister
	endpointsSynced cache.InformerSynced

	// proxyTransportDial specifies the dail function to creating unencrypted TCP connections.
	proxyTransportDial         *transport.DialHolder
	proxyCurrentCertKeyContent certKeyFn
	serviceResolver            ServiceResolver

	// To allow injection for testing.
	syncFn func(key string) error

	queue workqueue.TypedRateLimitingInterface[string]
	// map from service-namespace -> service-name -> api-service names
	cache map[string]map[string][]string
	// this lock protects operations one the above cache
	cacheLock sync.RWMutex

	// metrics registered into legacy registry
	metrics *metrics.Metrics
}

// New returns a new remote APIService AvailableConditionController.
func New(
	apiServiceInformer generatedClientInformersV1.APIServiceInformer,
	serviceInformer clientGoInformersCoreV1.ServiceInformer,
	endpointsInformer clientGoInformersCoreV1.EndpointsInformer,
	apiServiceClient generatedClientsetTypedV1.APIServicesGetter,
	proxyTransportDial *transport.DialHolder,
	proxyCurrentCertKeyContent certKeyFn,
	serviceResolver ServiceResolver,
	metrics *metrics.Metrics,
) (*AvailableConditionController, error) {
	c := &AvailableConditionController{
		apiServiceClient: apiServiceClient,
		apiServiceLister: apiServiceInformer.Lister(),
		serviceLister:    serviceInformer.Lister(),
		endpointsLister:  endpointsInformer.Lister(),
		serviceResolver:  serviceResolver,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			// We want a fairly tight requeue time.  The controller listens to the API, but because it relies on the routability of the
			// service network, it is possible for an external, non-watchable factor to affect availability.  This keeps
			// the maximum disruption time to a minimum, but it does prevent hot loops.
			workqueue.NewTypedItemExponentialFailureRateLimiter[string](5*time.Millisecond, 30*time.Second),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "RemoteAvailabilityController"},
		),
		proxyTransportDial:         proxyTransportDial,
		proxyCurrentCertKeyContent: proxyCurrentCertKeyContent,
		metrics:                    metrics,
	}

	// resync on this one because it is low cardinality and rechecking the actual discovery
	// allows us to detect health in a more timely fashion when network connectivity to
	// nodes is snipped, but the network still attempts to route there.  See
	// https://github.com/openshift/origin/issues/17159#issuecomment-341798063
	apiServiceHandler, _ := apiServiceInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.addAPIService,
			UpdateFunc: c.updateAPIService,
			DeleteFunc: c.deleteAPIService,
		},
		30*time.Second)
	c.apiServiceSynced = apiServiceHandler.HasSynced

	serviceHandler, _ := serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
		DeleteFunc: c.deleteService,
	})
	c.servicesSynced = serviceHandler.HasSynced

	endpointsHandler, _ := endpointsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addEndpoints,
		UpdateFunc: c.updateEndpoints,
		DeleteFunc: c.deleteEndpoints,
	})
	c.endpointsSynced = endpointsHandler.HasSynced

	c.syncFn = c.sync

	return c, nil
}

// processNextWorkItem deals with one key off the queue. It returns false when it's time to quit.
func (controller *AvailableConditionController) processNextWorkItem() bool {
	item, quited := controller.queue.Get()
	if quited {
		return false
	}
	defer controller.queue.Done(item)

	err := controller.syncFn(item)
	if err == nil {
		controller.queue.Forget(item)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("%v faield withe: %v", item, err))
	controller.queue.AddRateLimited(item)

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

	klog.Info("Starting RemoteAvailability controller")
	defer klog.Info("Shutting down RemoteAvailability controller")

	// This waits not just for the informers to sync, but for our handlers to be called; since the handlers are
	// three different ways of enqueueing the same thing, waiting for these permits the queue to maximally
	// de-duplicate the entries.
	if !controllers.WaitForCacheSync("RemoteAvailability", stopCh, controller.apiServiceSynced) {
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
		controller.queue.Forget(key)
		return nil
	}
	if err != nil {
		return err
	}

	if rawAPIService.Spec.Service == nil {
		return nil
	}

	apiService := rawAPIService.DeepCopy()
	// if particular transport was specified, use that otherwise build one construct a http client that will ignore
	// TLS verification (if someone owns the network and messes with your status that's out so bad) and sets very
	// short timeout.  This is the best effort GET that provides no additional information.
	transportConfig := &transport.Config{
		TLS: transport.TLSConfig{
			Insecure: true,
		},
		DialHolder: controller.proxyTransportDial,
	}

	if controller.proxyCurrentCertKeyContent != nil {
		proxyClientCert, proxyClientKey := controller.proxyCurrentCertKeyContent()
		transportConfig.TLS.CertData = proxyClientCert
		transportConfig.TLS.KeyData = proxyClientKey
	}
	restTransport, err := transport.New(transportConfig)
	if err != nil {
		return err
	}
	discoveryClient := &http.Client{
		Transport: restTransport,
		// the request should happen quickly.
		Timeout:       5 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}

	availableCondition := apiregistrationv1.APIServiceCondition{
		Type:               apiregistrationv1.Available,
		Status:             apiregistrationv1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
	}

	service, err := controller.serviceLister.Services(apiService.Spec.Service.Namespace).Get(apiService.Spec.Service.Name)
	if errors.IsNotFound(err) {
		availableCondition.Status = apiregistrationv1.ConditionFalse
		availableCondition.Reason = "ServiceNotFound"
		availableCondition.Message = fmt.Sprintf("service/%s in %q is not present", apiService.Spec.Service.Name,
			apiService.Spec.Service.Namespace)
		apiregistrationv1helper.SetAPIServiceCondition(apiService, availableCondition)
		_, err := controller.updateAPIServiceStatus(rawAPIService, apiService)
		return err
	} else if err != nil {
		availableCondition.Status = apiregistrationv1.ConditionFalse
		availableCondition.Reason = "ServiceAccessError"
		availableCondition.Message = fmt.Sprintf("service/%s in %q cannot be checked due to: %v",
			apiService.Spec.Service.Name, apiService.Spec.Service.Namespace, err)
		apiregistrationv1helper.SetAPIServiceCondition(apiService, availableCondition)
		_, err := controller.updateAPIServiceStatus(rawAPIService, apiService)
		return err
	}

	if service.Spec.Type == kubeAPICorev1.ServiceTypeClusterIP {
		// if we hava a cluster IP service, it must be listening on configured port and can check that
		servicePort := apiService.Spec.Service.Port
		portName := ""
		foundPort := false
		for _, port := range service.Spec.Ports {
			if port.Port == *servicePort {
				foundPort = true
				portName = port.Name
				break
			}
		}

		if !foundPort {
			availableCondition.Status = apiregistrationv1.ConditionFalse
			availableCondition.Reason = "ServicePortError"
			availableCondition.Message = fmt.Sprintf("service/%s in %q is not listening on port %d",
				apiService.Spec.Service.Name, apiService.Spec.Service.Namespace, *servicePort)
			apiregistrationv1helper.SetAPIServiceCondition(apiService, availableCondition)
			_, err := controller.updateAPIServiceStatus(rawAPIService, apiService)
			return err
		}

		endpoints, err := controller.endpointsLister.Endpoints(apiService.Spec.Service.Namespace).Get(apiService.Spec.Service.Name)
		if errors.IsNotFound(err) {
			availableCondition.Status = apiregistrationv1.ConditionFalse
			availableCondition.Reason = "EndpointsNotFound"
			availableCondition.Message = fmt.Sprintf("cannot find endpoints for service/%s in %q", apiService.Spec.Service.Name,
				apiService.Spec.Service.Namespace)
			apiregistrationv1helper.SetAPIServiceCondition(apiService, availableCondition)
			_, err := controller.updateAPIServiceStatus(rawAPIService, apiService)
			return err
		} else if err != nil {
			availableCondition.Status = apiregistrationv1.ConditionFalse
			availableCondition.Reason = "EndpointsAccessError"
			availableCondition.Message = fmt.Sprintf("service/%s in %q cannot be checked due to: %v",
				apiService.Spec.Service.Name, apiService.Spec.Service.Namespace, err)
			apiregistrationv1helper.SetAPIServiceCondition(apiService, availableCondition)
			_, err := controller.updateAPIServiceStatus(rawAPIService, apiService)
			return err
		}
		hasActiveEndpoints := false
	findEndpointsBySubsetsLoopOuter:
		for _, subset := range endpoints.Subsets {
			if len(subset.Addresses) == 0 {
				continue
			}
			for _, endpointPort := range subset.Ports {
				if endpointPort.Name == portName {
					hasActiveEndpoints = true
					break findEndpointsBySubsetsLoopOuter
				}
			}
		}
		if !hasActiveEndpoints {
			availableCondition.Status = apiregistrationv1.ConditionFalse
			availableCondition.Reason = "MissingEndpoints"
			availableCondition.Message = fmt.Sprintf("endpoints for service/%s in %q have no addresses with port name %q", apiService.Spec.Service.Name, apiService.Spec.Service.Namespace, portName)
			apiregistrationv1helper.SetAPIServiceCondition(apiService, availableCondition)
			_, err := controller.updateAPIServiceStatus(rawAPIService, apiService)
			return err
		}
	}

	// actually try to hit the discovery endpoint when it isn't local and when we are routing a service.
	if apiService.Spec.Service != nil && controller.serviceResolver != nil {
		attempts := 5
		results := make(chan error, attempts)
		for i := 0; i < attempts; i++ {
			go func() {
				discoveryURL, err := controller.serviceResolver.ResolveEndpoint(apiService.Spec.Service.Namespace,
					apiService.Spec.Service.Name, *apiService.Spec.Service.Port)
				if err != nil {
					results <- err
					return
				}
				// render legacyAPIService health check path when it is delegated to a service
				if apiService.Name == "v1." {
					discoveryURL.Path = "/api/" + apiService.Spec.Version
				} else {
					discoveryURL.Path = "/apis/" + apiService.Spec.Group + "/" + apiService.Spec.Version
				}

				errCh := make(chan error, 1)
				go func() {
					// be sure to check a URL that the aggregated API server is required to serve
					newRequest, err := http.NewRequest(http.MethodGet, discoveryURL.String(), nil)
					if err != nil {
						errCh <- err
						return
					}

					// setting the system-masters identity ensures that we will always hava access rights
					transport.SetAuthProxyHeaders(newRequest, "system:kube-aggregator", "",
						[]string{"system:masters"}, nil)
					response, err := discoveryClient.Do(newRequest)
					if response != nil {
						_ = response.Body.Close()
						// we should always been in the 200s or 300s
						if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultiStatus {
							errCh <- fmt.Errorf("bad status from %v: %d", discoveryURL, response.StatusCode)
							return
						}
					}
					errCh <- err
				}()

				select {
				case err = <-errCh:
					if err != nil {
						results <- fmt.Errorf("failing or missing response from %v: %w", discoveryURL, err)
						return
					}

				// we had trouble with slow dial and DNS responses causing us to wait too long.
				// we added this insurance
				case <-time.After(6 * time.Second):
					results <- fmt.Errorf("timed out wait for %v", discoveryURL)
					return
				}

				results <- nil
			}()
		}

		var lastError error
		for i := 0; i < attempts; i++ {
			lastError = <-results
			// if we had at least one success, we are successful overall and we can return new
			if lastError == nil {
				break
			}
		}

		if lastError != nil {
			availableCondition.Status = apiregistrationv1.ConditionFalse
			availableCondition.Reason = "FailedDiscoveryCheck"
			availableCondition.Message = lastError.Error()
			apiregistrationv1helper.SetAPIServiceCondition(apiService, availableCondition)
			_, updateErr := controller.updateAPIServiceStatus(rawAPIService, apiService)
			// force a requeue to make it very obvious that this will be retried at some point in the future
			// along with other requeues done via service change, endpoint change, and resync
			return updateErr
		}
	}

	availableCondition.Reason = "Passed"
	availableCondition.Message = "all checks passed"
	apiregistrationv1helper.SetAPIServiceCondition(apiService, availableCondition)
	_, err = controller.updateAPIServiceStatus(rawAPIService, apiService)
	return err
}

// if the service/endpoint handler wins the race against the cache rebuilding, it may queue a no-longer-relevant apiservice
// (which will get processed an extra time - this doesn't matter),
// and miss a newly relevant apiservice (which will get queued by the apiservice handler)
func (controller *AvailableConditionController) rebuildAPIServiceCache() {
	apiServices, _ := controller.apiServiceLister.List(labels.Everything())
	newCache := map[string]map[string][]string{}
	for _, apiService := range apiServices {
		if apiService.Spec.Service == nil {
			continue
		}
		if newCache[apiService.Spec.Service.Namespace] == nil {
			newCache[apiService.Spec.Service.Namespace] = map[string][]string{}
		}
		newCache[apiService.Spec.Service.Namespace][apiService.Spec.Service.Name] =
			append(newCache[apiService.Spec.Service.Namespace][apiService.Spec.Service.Name], apiService.Name)
	}

	controller.cacheLock.Lock()
	defer controller.cacheLock.Unlock()
	controller.cache = newCache
}

func (controller *AvailableConditionController) addAPIService(obj interface{}) {
	apiService := obj.(*apiregistrationv1.APIService)
	klog.V(4).Infof("Adding %s", apiService.Name)
	if apiService.Spec.Service != nil {
		controller.rebuildAPIServiceCache()
	}
	controller.queue.Add(apiService.Name)
}

func (controller *AvailableConditionController) updateAPIService(oldObj, newObj interface{}) {
	newAPIService := newObj.(*apiregistrationv1.APIService)
	rawAPIService := oldObj.(*apiregistrationv1.APIService)
	klog.V(4).Infof("Updating %s", rawAPIService.Name)
	if !reflect.DeepEqual(newAPIService.Spec.Service, rawAPIService.Spec.Service) {
		controller.rebuildAPIServiceCache()
	}
	controller.queue.Add(rawAPIService.Name)
}

func (controller *AvailableConditionController) deleteAPIService(obj interface{}) {
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
	if castObj.Spec.Service != nil {
		controller.rebuildAPIServiceCache()
	}
	controller.queue.Add(castObj.Name)
}

func (controller *AvailableConditionController) getAPIServicesFor(obj runtime.Object) []string {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return nil
	}
	controller.cacheLock.RLock()
	defer controller.cacheLock.RUnlock()
	return controller.cache[metadata.GetNamespace()][metadata.GetName()]
}

// TODO, think of a way to avoid checking on every service manipulation

func (controller *AvailableConditionController) addService(obj interface{}) {
	for _, apiService := range controller.getAPIServicesFor(obj.(*kubeAPICorev1.Service)) {
		controller.queue.Add(apiService)
	}
}

func (controller *AvailableConditionController) updateService(oldObj, _ interface{}) {
	for _, apiService := range controller.getAPIServicesFor(oldObj.(*kubeAPICorev1.Service)) {
		controller.queue.Add(apiService)
	}
}

func (controller *AvailableConditionController) deleteService(obj interface{}) {
	castObj, ok := obj.(*kubeAPICorev1.Service)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		castObj, ok = tombstone.Obj.(*kubeAPICorev1.Service)
		if !ok {
			klog.Errorf("Tombstone contained object that is not expected %#v", obj)
			return
		}
	}
	for _, apiService := range controller.getAPIServicesFor(castObj) {
		controller.queue.Add(apiService)
	}
}

func (controller *AvailableConditionController) addEndpoints(obj interface{}) {
	for _, endpoint := range controller.getAPIServicesFor(obj.(*kubeAPICorev1.Endpoints)) {
		controller.queue.Add(endpoint)
	}
}

func (controller *AvailableConditionController) updateEndpoints(oldObj, _ interface{}) {
	for _, endpoint := range controller.getAPIServicesFor(oldObj.(*kubeAPICorev1.Endpoints)) {
		controller.queue.Add(endpoint)
	}
}

func (controller *AvailableConditionController) deleteEndpoints(obj interface{}) {
	castObj, ok := obj.(*kubeAPICorev1.Endpoints)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		castObj, ok = tombstone.Obj.(*kubeAPICorev1.Endpoints)
		if !ok {
			klog.Errorf("Tombstone contained object that is not expected %#v", obj)
			return
		}
	}
	for _, apiService := range controller.getAPIServicesFor(castObj) {
		controller.queue.Add(apiService)
	}
}
