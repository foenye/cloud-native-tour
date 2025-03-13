package apiserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration"
	apiregistrationv1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1helper "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1/helper"
	apiDiscoveryV2 "k8s.io/api/apidiscovery/v2"
	apiDiscoveryV2beta1 "k8s.io/api/apidiscovery/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	apiDiscoveryV2Conversion "k8s.io/apiserver/pkg/apis/apidiscovery/v2"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints"
	endpointsDiscoveryAggregated "k8s.io/apiserver/pkg/endpoints/discovery/aggregated"
	endpointsRequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/util/responsewriter"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"net/http"
	"sync"
	"time"
)

var APIRegistrationGroupVersion = metav1.GroupVersion{Group: apiregistration.GroupName, Version: "v1"}

// APIRegistrationGroupPriority maximum is 20000. Set to higher than that so apiregistration always is listed
// first (mirrors v1 discovery behavior)
var APIRegistrationGroupPriority = 20001

// Aggregated discovery content-type GVK.
var v2Beta1GVK = schema.GroupVersionKind{
	Group:   "apidiscovery.k8s.io",
	Version: "v2beta1",
	Kind:    "APIGroupDiscoveryList",
}
var v2GVK = schema.GroupVersionKind{
	Group:   "apidiscovery.k8s.io",
	Version: "v2",
	Kind:    "APIGroupDiscoveryList",
}

type DiscoveryAggregationController interface {
	// AddAPIService adds or updates an api service from the aggregated discovery controller's knowledge base.
	// Thread-safe.
	AddAPIService(apiService *apiregistrationv1.APIService, httpHandler http.Handler)
	// RemoveAPIService removes an api service from the aggregated discovery controller's knowledge bank.
	// Thread-safe.
	RemoveAPIService(apiServiceName string)
	// Run spawns a worker which waits for added/updated api services and updates the unified discovery document
	// by contacting the aggregated api services.
	Run(stopCh <-chan struct{}, discoverySyncCh chan<- struct{})
}

var _ fmt.Stringer = serviceKey{}

// serviceKey version of Service/Spec with relevant fields for use a cache key.
type serviceKey struct {
	Namespace string
	Name      string
	Port      int32
}

func newServiceKey(serviceReference apiregistrationv1.ServiceReference) serviceKey {
	// Docs say. Defaults to 443 for compatibility reasons.
	// BETA: Should this be a shared constant to avoid drifting with the implementation?
	port := int32(443)
	if serviceReference.Port != nil {
		port = *serviceReference.Port
	}
	return serviceKey{
		Namespace: serviceReference.Namespace,
		Name:      serviceReference.Name,
		Port:      port,
	}
}

// String human-readable String representation used for logs.
func (key serviceKey) String() string {
	return fmt.Sprintf("%v/%v:%v", key.Namespace, key.Name, key.Port)
}

type groupVersionInfo struct {
	// lastMarkedDirty Date this APIService was marked dirty.
	// Guaranteed to be a time greater than the more recent time the APIService was known to be modified.
	//
	// Used for request deduplication to ensure the data used to reconcile each APIService was retrieved after the
	// time of the APIService change:
	// real_apiservice_change_time < groupVersionInfo.lastMarkedDirty < cachedResult.lastUpdated < real_document_fresh_time
	//
	// This ensures that if the APIService was changed after the last cached entry was sorted, the discovery document
	// always be re-fetched.
	lastMarkedDirty time.Time

	// service reference of this GroupVersion. This identifies the service which describes how to contact the server
	// responsible for this GroupVersion.
	service serviceKey

	// groupPriority describes the priority of the APIService's group for sorting.
	groupPriority int

	// versionPriority describes the priority of the APIService's version for sorting.
	versionPriority int

	// Method for contacting the service.
	httpHandler http.Handler
}

type cachedResult struct {
	// discovery is cached discovery document for this service map from group name to version name to
	discovery map[metav1.GroupVersion]apiDiscoveryV2.APIVersionDiscovery
	// eTag is e-tag hash of the cached discoveryDocument.
	eTag string
	// lastUpdated guaranteed to be a time less than the time the sever responded with the discovery data.
	lastUpdated time.Time
}

type discoveryManagerImplementation interface {
	getInfoForAPIService(name string) (groupVersionInfo, bool)
	setInfoForAPIService(name string, result *groupVersionInfo) (oldValueIfExisted *groupVersionInfo)
	getCacheEntryForService(key serviceKey) (cachedResult, bool)
	setCacheEntryForService(key serviceKey, result cachedResult)
	fetchFreshDiscoveryForService(gv metav1.GroupVersion, info groupVersionInfo) (*cachedResult, error)
	removeUnusedServices()
	getAPIServiceKeys() []string
	syncAPIService(apiServiceName string) error

	DiscoveryAggregationController
}

var _ discoveryManagerImplementation = &discoveryManager{}

type discoveryManager struct {
	// servicesLock Locks api services.
	servicesLock sync.RWMutex

	// apiServices is map form APIService's name (or a unique string of local services) to information about
	// contacting that API service
	apiServices map[string]groupVersionInfo

	// resultsLock is Locks cachedResults
	resultsLock sync.RWMutex

	// cachedResults is map from APIService.Spec.Service to the previously fetched value
	// (Note that many APIServices might use the same APIService.Spec.Service)
	cachedResults map[serviceKey]cachedResult

	// dirtyAPIServiceQueue is queue of dirty apiServiceKey which need to be refreshed.
	// It is important that the reconciler for this queue does not excessively contact the API server if a key was
	// enqueued before the server was last contacted.
	dirtyAPIServiceQueue workqueue.TypedRateLimitingInterface[string]

	// mergedDiscoveryHandler merged handler which stores all known group versions.
	mergedDiscoveryHandler endpointsDiscoveryAggregated.ResourceManager

	// codecs is the serializer used for decoding aggregated api service responses.
	codecs serializer.CodecFactory
}

func NewDiscoveryManager(resourceManager endpointsDiscoveryAggregated.ResourceManager) DiscoveryAggregationController {
	discoveryScheme := runtime.NewScheme()
	utilruntime.Must(apiDiscoveryV2.AddToScheme(discoveryScheme))
	utilruntime.Must(apiDiscoveryV2beta1.AddToScheme(discoveryScheme))
	// Register conversion for api discovery
	utilruntime.Must(apiDiscoveryV2Conversion.AddToScheme(discoveryScheme))
	codecs := serializer.NewCodecFactory(discoveryScheme)

	return &discoveryManager{
		mergedDiscoveryHandler: resourceManager,
		apiServices:            make(map[string]groupVersionInfo),
		cachedResults:          make(map[serviceKey]cachedResult),
		dirtyAPIServiceQueue: workqueue.NewTypedRateLimitingQueueWithConfig[string](
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "discovery-manager"},
		),
		codecs: codecs,
	}

}

func (manager *discoveryManager) getInfoForAPIService(name string) (groupVersionInfo, bool) {
	manager.servicesLock.RLock()
	defer manager.servicesLock.RUnlock()

	info, exists := manager.apiServices[name]
	return info, exists
}

func (manager *discoveryManager) setInfoForAPIService(name string, result *groupVersionInfo) (oldValueIfExisted *groupVersionInfo) {
	manager.servicesLock.Lock()
	defer manager.servicesLock.Unlock()

	if oldValue, exists := manager.apiServices[name]; exists {
		oldValueIfExisted = &oldValue
	}

	if result != nil {
		manager.apiServices[name] = *result
	} else {
		delete(manager.apiServices, name)
	}

	return oldValueIfExisted
}

func (manager *discoveryManager) getCacheEntryForService(key serviceKey) (cachedResult, bool) {
	manager.resultsLock.RLock()
	defer manager.resultsLock.RUnlock()

	result, exists := manager.cachedResults[key]
	return result, exists
}

func (manager *discoveryManager) setCacheEntryForService(key serviceKey, result cachedResult) {
	manager.resultsLock.Lock()
	defer manager.resultsLock.Unlock()

	manager.cachedResults[key] = result
}

// fetchFreshDiscoveryForService returns discovery data for the given apiservice.
// Caches the result.
// Returns the cached result if it is retrieved after the apiservice was last marked dirty
// If there was an error in fetching, returns the stale cached result if it exists, and a non-nil error
// If the result is current, returns nil error and non-nil result
func (manager *discoveryManager) fetchFreshDiscoveryForService(gv metav1.GroupVersion, info groupVersionInfo) (*cachedResult, error) {
	// Lockup last cached result for this APIService's service.
	cached, exists := manager.getCacheEntryForService(info.service)
	// If entry exists and was updated after the given time, just stop now.
	if exists && cached.lastUpdated.After(info.lastMarkedDirty) {
		return &cached, nil
	}

	// If we hava a handler to contact the server fot this APIService, and the cache entry is too old to use, refresh
	// the cache entry now.
	httpHandler := http.TimeoutHandler(info.httpHandler, 5*time.Second, "request timed out")
	request, err := http.NewRequest(http.MethodGet, "/apis", nil)
	if err != nil {
		return &cached, fmt.Errorf("failed to create http.Request: %v", err)
	}

	// Apply aggregator user to request
	request = request.WithContext(endpointsRequest.WithUser(request.Context(), &user.DefaultInfo{
		Name:   "system:kube-aggregator",
		Groups: []string{"system:masters"},
	}))
	request = request.WithContext(endpointsRequest.WithRequestInfo(request.Context(), &endpointsRequest.RequestInfo{
		Path:              request.URL.Path,
		IsResourceRequest: false,
	}))
	request.Header.Add("Accept", discovery.AcceptV2+","+discovery.AcceptV2Beta1)

	if exists && len(cached.eTag) > 0 {
		request.Header.Add("If-None-Match", cached.eTag)
	}

	// Important that the time recorded in the data's "lastUpdated" is conservatively from BEFORE the request is
	// dispatched so that lastUpdated can be used to de-duplicate requests.
	now := time.Now()
	response := responsewriter.NewInMemoryResponseWriter()
	httpHandler.ServeHTTP(response, request)

	isV2Beta1GVK, _ := discovery.ContentTypeIsGVK(response.Header().Get("Content-Type"), v2Beta1GVK)
	isV2GVK, _ := discovery.ContentTypeIsGVK(response.Header().Get("Content-Type"), v2GVK)

	switch {
	case response.RespCode() == http.StatusNotModified:
		// Keep old entry, update timestamp
		cached = cachedResult{
			discovery:   cached.discovery,
			eTag:        cached.eTag,
			lastUpdated: now,
		}

		manager.setCacheEntryForService(info.service, cached)
		return &cached, nil
	case response.RespCode() == http.StatusOK && (isV2Beta1GVK || isV2GVK):
		parsed := &apiDiscoveryV2.APIGroupDiscoveryList{}
		if err := runtime.DecodeInto(manager.codecs.UniversalDecoder(), response.Data(), parsed); err != nil {
			return nil, err
		}

		klog.V(3).Infof("DiscoveryManager: Successfully downloaded discovery for %s", info.service.String())

		// Convert discovery info into a map for convenient lookup later
		discoveryMap := map[metav1.GroupVersion]apiDiscoveryV2.APIVersionDiscovery{}
		for _, groupDiscovery := range parsed.Items {
			for _, versionDiscovery := range groupDiscovery.Versions {
				discoveryMap[metav1.GroupVersion{Group: groupDiscovery.Name, Version: versionDiscovery.Version}] =
					versionDiscovery
				for idx := range versionDiscovery.Resources {
					// avoid nil panics in v0.26.0-v0.26.3 client-go clients
					// see https://github.com/kubernetes/kubernetes/issues/118361
					if versionDiscovery.Resources[idx].ResponseKind == nil {
						versionDiscovery.Resources[idx].ResponseKind = &metav1.GroupVersionKind{}
					}

					for subIdx := range versionDiscovery.Resources[idx].Subresources {
						if versionDiscovery.Resources[idx].Subresources[subIdx].ResponseKind == nil {
							versionDiscovery.Resources[idx].Subresources[subIdx].ResponseKind = &metav1.GroupVersionKind{}
						}
					}
				}
			}
		}

		// Save cached result
		cached = cachedResult{
			discovery:   discoveryMap,
			eTag:        response.Header().Get("Etag"),
			lastUpdated: now,
		}
		manager.setCacheEntryForService(info.service, cached)
		return &cached, nil

	default:
		// Could not get acceptable response for Aggregated Discovery.
		// Fall back to legacy discovery information
		if len(gv.Version) == 0 {
			return nil, errors.New("not found")
		}

		var path string
		if len(gv.Group) == 0 {
			path = "/api/" + gv.Version
		} else {
			path = "/apis/" + gv.Group + "/" + gv.Version
		}

		request, _ := http.NewRequest(http.MethodGet, path, nil)
		request = request.WithContext(endpointsRequest.WithUser(request.Context(), &user.DefaultInfo{
			Name: "system:aggregator",
		}))

		//request.Header.Add("Accept", runtime.ContentTypeProtobuf)
		request.Header.Add("Accept", runtime.ContentTypeJSON)

		if exists && len(cached.eTag) > 0 {
			request.Header.Add("If-None-Match", cached.eTag)
		}
		response := responsewriter.NewInMemoryResponseWriter()
		httpHandler.ServeHTTP(response, request)

		if response.RespCode() != http.StatusOK {
			return nil, fmt.Errorf("failed to download legacy discovery for %s: %v", path, response.String())
		}

		parsed := &metav1.APIResourceList{}
		if err := runtime.DecodeInto(manager.codecs.UniversalDecoder(), response.Data(), parsed); err != nil {
			return nil, err
		}

		// Create a discoveryMap with single group-version
		resourceDiscoveries, err := endpoints.ConvertGroupVersionIntoToDiscovery(parsed.APIResources)
		if err != nil {
			return nil, err
		}
		klog.V(3).Infof("DiscoveryManager: Successfully downloaded legacy discovery for %s", info.service.String())

		discoveryMap := map[metav1.GroupVersion]apiDiscoveryV2.APIVersionDiscovery{
			gv: {
				Version:   gv.Version,
				Resources: resourceDiscoveries,
			},
		}

		cached = cachedResult{
			discovery:   discoveryMap,
			lastUpdated: now,
		}

		// Do not save the resolve as the legacy fallback only fetches one group version and an API service may
		// serve multiple group versions.
		return &cached, nil

	}
}

// removeUnusedServices takes a snapshot of all currently used services by known APIServices and purges the cache
// entries of chose not present in the snapshot.
func (manager *discoveryManager) removeUnusedServices() {
	usedServiceKeys := sets.Set[serviceKey]{}
	func() {
		manager.servicesLock.Lock()
		defer manager.servicesLock.Unlock()

		// Mark all non-local APIServices as dirty.
		for _, info := range manager.apiServices {
			usedServiceKeys.Insert(info.service)
		}
	}()

	// Avoids double lock. It is okay if a service is added/removed between these functions.
	// This is just a cache and that should be infrequent.

	func() {
		manager.resultsLock.Lock()
		defer manager.resultsLock.Unlock()

		for key := range manager.cachedResults {
			if !usedServiceKeys.Has(key) {
				delete(manager.cachedResults, key)
			}
		}
	}()
}

func (manager *discoveryManager) getAPIServiceKeys() []string {
	manager.servicesLock.RLock()
	defer manager.servicesLock.RUnlock()

	var keys []string
	for key := range manager.apiServices {
		keys = append(keys, key)
	}

	return keys
}

// syncAPIService try to sync a single APIService.
func (manager *discoveryManager) syncAPIService(apiServiceName string) error {
	info, exists := manager.getInfoForAPIService(apiServiceName)

	groupVersion := apiregistrationv1helper.APIServiceNameToGroupVersion(apiServiceName)
	mergedGroupVersion := metav1.GroupVersion{Group: groupVersion.Group, Version: groupVersion.Version}

	if !exists {
		// API service was removed. remove it from merged discovery
		manager.mergedDiscoveryHandler.RemoveGroupVersion(mergedGroupVersion)
		return nil
	}

	// Lookup last cached result from this APIService's service
	cached, err := manager.fetchFreshDiscoveryForService(mergedGroupVersion, info)

	var entry apiDiscoveryV2.APIVersionDiscovery

	// Extract the APIService's specific resource information form the group version
	if cached == nil {
		// There was an error fetching discovery for this APIService, and there is nothing in the cache for this GV.
		//
		// Just to use empty GV to mark that GV exists, but no resources.  Also mark that it is stale to indicate
		// the fetched failed.
		// TODO: Maybe also stick in a status for the version the error?
		entry = apiDiscoveryV2.APIVersionDiscovery{Version: groupVersion.Version}
	} else {
		// Find our specific GV within the discovery document
		entry, exists = cached.discovery[mergedGroupVersion]
		if exists {
			// The stale/fresh entry has our GV, so we can include it in the doc
		} else {
			// Successfully fetched discovery information form the server, but the server did not include this GV?
			entry = apiDiscoveryV2.APIVersionDiscovery{Version: groupVersion.Version}
		}
	}

	// The entry's staleness depends upon if `fetchFreshDiscoveryForService` returned an error or not.
	if err == nil {
		entry.Freshness = apiDiscoveryV2.DiscoveryFreshnessCurrent
	} else {
		entry.Freshness = apiDiscoveryV2.DiscoveryFreshnessStale
	}

	manager.mergedDiscoveryHandler.AddGroupVersion(groupVersion.Group, entry)
	manager.mergedDiscoveryHandler.SetGroupVersionPriority(metav1.GroupVersion(groupVersion), info.groupPriority,
		info.versionPriority)
	return nil
}

// AddAPIService adds an APIService to be tracked by the discovery manger. If the APIServer is already known.
func (manager *discoveryManager) AddAPIService(apiService *apiregistrationv1.APIService, httpHandler http.Handler) {
	// If service is nil then its information is contained by a local APIService which is has already been added
	// to manager.
	if apiService.Spec.Service == nil {
		return
	}

	// Add or update APIService record and mark it as dirty.
	manager.setInfoForAPIService(apiService.Name, &groupVersionInfo{
		groupPriority:   int(apiService.Spec.GroupPriorityMinimum),
		versionPriority: int(apiService.Spec.VersionPriority),
		httpHandler:     httpHandler,
		lastMarkedDirty: time.Now(),
		service:         newServiceKey(*apiService.Spec.Service),
	})
	manager.removeUnusedServices()
	manager.dirtyAPIServiceQueue.Add(apiService.Name)
}

func (manager *discoveryManager) RemoveAPIService(apiServiceName string) {
	if manager.setInfoForAPIService(apiServiceName, nil) != nil {
		// Mark dirty if there was actually something deleted.
		manager.removeUnusedServices()
		manager.dirtyAPIServiceQueue.Add(apiServiceName)
	}
}

// Run spawns a goroutine which waits for added/updated APIService and updates the discovery document accordingly
func (manager *discoveryManager) Run(stopCh <-chan struct{}, discoverySyncCh chan<- struct{}) {
	klog.Info("Starting ResourceDiscoveryManager")

	// Shutdown the queue since stopCh was signalled
	defer manager.dirtyAPIServiceQueue.ShutDown()

	// Ensure that apiregistration.eonvon.github.io is the first group in the discovery group.
	manager.mergedDiscoveryHandler.WithSource(endpointsDiscoveryAggregated.BuiltinSource).SetGroupVersionPriority(
		APIRegistrationGroupVersion, APIRegistrationGroupPriority, 0)

	// Ensure that all APIServices are present before readiness check succeeds.
	var wg sync.WaitGroup
	// Iterate on a copy of the keys to be thread safe with syncAPIService
	keys := manager.getAPIServiceKeys()

	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			// If an error was returned, ths api service will still hava been added but marked as stale.
			// Ignore the return value here.
			_ = manager.syncAPIService(k)
		}(key)
	}
	wg.Wait()

	if discoverySyncCh != nil {
		close(discoverySyncCh)
	}

	// Spawn workers
	// These workers wait for APIServices to be marked dirty.
	// Worker ensures the cached discovery document hosted by the ServiceReference of the APIService is at least as
	// fresh as the APIService, then includes the APIService's GV into the merged document.
	for i := 0; i < 2; i++ {
		go func() {
			for {
				next, shutdown := manager.dirtyAPIServiceQueue.Get()
				if shutdown {
					return
				}
				func() {
					defer manager.dirtyAPIServiceQueue.Done(next)

					if err := manager.syncAPIService(next); err != nil {
						manager.dirtyAPIServiceQueue.AddRateLimited(next)
					} else {
						manager.dirtyAPIServiceQueue.Forget(next)
					}
				}()
			}
		}()
	}
	_ = wait.PollUntilContextCancel(wait.ContextForChannel(stopCh), time.Minute, true, func(ctx context.Context) (done bool, err error) {
		manager.servicesLock.Lock()
		defer manager.servicesLock.Unlock()

		now := time.Now()

		// Mark all non-local APIServices as dirty
		for key, info := range manager.apiServices {
			info.lastMarkedDirty = now
			manager.apiServices[key] = info
			manager.dirtyAPIServiceQueue.Add(key)
		}
		return false, nil
	})
}
