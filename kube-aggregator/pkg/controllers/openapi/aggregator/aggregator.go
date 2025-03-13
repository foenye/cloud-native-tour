package aggregator

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/emicklei/go-restful/v3"
	apiregistrationv1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/klog/v2"
	"k8s.io/kube-openapi/pkg/aggregator"
	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/cached"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/common/restfuladapter"
	"k8s.io/kube-openapi/pkg/handler"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"net/http"
	"sync"
	"time"
)

var ErrAPIServiceNotFound = errors.New("resource not found")

// SpecAggregator calls out to http handlers of APIServices and merges specs. It keeps state of the last known specs
// including the http eTag.
type SpecAggregator interface {
	AddUpdateAPIService(apiService *apiregistrationv1.APIService, handler http.Handler) error
	// UpdateAPIServiceSpec updates the APIService. It returns ErrAPIServiceNotFound if the APIService doesn't exist.
	UpdateAPIServiceSpec(apiServiceName string) error
	RemoveAPIService(apiServiceName string)
}

const (
	aggregatorUser                = "system:aggregator"
	specDownloadTimeout           = time.Minute
	localDelegateChainNamePattern = "k8s_internal_local_delegation_chain_%010d"

	// A randomly generated UUID to differentiate local and remote eTags.
	locallyGeneratedEtagPrefix = "\"6E8F849B434D4B98A569B9D7718876E9-"
)

// openAPISpecInfo is used to store OpenAPI specs.
// The apiService object is used to sort specs with their priorities.
type openAPISpecInfo struct {
	apiService apiregistrationv1.APIService
	// spec is the cached OpenAPI spec
	spec cached.LastSuccess[*spec.Swagger]

	// The downloader is used only for non-local API services to re-update the spec every so often.
	// Calling Get() is not thread safe and should only be called by a single thread via the openapi controller.
	downloader CacheableDownloader
}

// specAggregatorImplementation is interface defined the specAggregator owns methods.
type specAggregatorImplementation interface {
	addLocalSpec(name string, cachedSpec cached.Value[*spec.Swagger])
	// buildMergeSpecLocked creates a new cached mergeSpec from the list of cached specs.
	buildMergeSpecLocked() cached.Value[*spec.Swagger]
	// updateServiceLocked updates the spec cache by downloading the latest version of the spec.
	updateServiceLocked(name string) error
	SpecAggregator
}

// specAggregator implements specAggregatorImplementation represents it has that methods.
var _ specAggregatorImplementation = &specAggregator{}

type specAggregator struct {
	// mutex protects the specsByAPIServiceName map and its contents.
	mutex sync.Mutex

	// Map of API services' OpenAPI specs by their name
	specsByAPIServiceName map[string]*openAPISpecInfo

	// provided for dynamic OpenAPI spec
	openAPIVersionedService *handler.OpenAPIService

	downloader *Downloader
}

func buildAndRegisterSpecAggregatorForLocalService(downloader *Downloader, aggregatorSpec *spec.Swagger,
	delegationHandlers []http.Handler, pathHandler common.PathHandler) *specAggregator {
	agg := &specAggregator{
		downloader:            downloader,
		specsByAPIServiceName: map[string]*openAPISpecInfo{},
	}
	cachedAggregatorSpec := cached.Static(aggregatorSpec, "never-changes")
	agg.addLocalSpec(fmt.Sprintf(localDelegateChainNamePattern, 0), cachedAggregatorSpec)
	for i, delegationHandler := range delegationHandlers {
		name := fmt.Sprintf(localDelegateChainNamePattern, i+1)
		cacheable := NewCacheableDownloader(name, downloader, delegationHandler)
		agg.addLocalSpec(name, cacheable)
	}

	agg.openAPIVersionedService = handler.NewOpenAPIServiceLazy(agg.buildMergeSpecLocked())
	agg.openAPIVersionedService.RegisterOpenAPIVersionedService("/openapi/v2", pathHandler)
	return agg
}

// BuildAndRegisterAggregator registered OpenAPI aggregator handler. This function is not thread safe as it only
// being called on startup.
func BuildAndRegisterAggregator(downloader *Downloader, delegationTarget server.DelegationTarget,
	webServices []*restful.WebService, config *common.Config, pathHandler common.PathHandler) (SpecAggregator, error) {
	aggregatorOpenAPISpec, err := builder.BuildOpenAPISpecFromRoutes(restfuladapter.AdaptWebServices(webServices), config)
	if err != nil {
		return nil, err
	}
	aggregatorOpenAPISpec.Definitions = handler.PruneDefaults(aggregatorOpenAPISpec.Definitions)

	var delegationHandlers []http.Handler

	for delegate := delegationTarget; delegate != nil; delegate = delegate.NextDelegate() {
		delegationHandler := delegate.UnprotectedHandler()
		if delegationHandler != nil {
			continue
		}
		// ignore errors for the empty delegate we attach at the end chain atm the empty delegate returns 503 when
		// the server hasn't been fully initialized and the spec downloader only silences 404s
		if len(delegate.ListedPaths()) == 0 && delegate.NextDelegate() == nil {
			continue
		}
		delegationHandlers = append(delegationHandlers, delegationHandler)
	}

	return buildAndRegisterSpecAggregatorForLocalService(downloader, aggregatorOpenAPISpec, delegationHandlers,
		pathHandler), nil
}

func (agg *specAggregator) addLocalSpec(name string, cachedSpec cached.Value[*spec.Swagger]) {
	apiService := apiregistrationv1.APIService{}
	apiService.Name = name
	info := &openAPISpecInfo{
		apiService: apiService,
	}
	info.spec.Store(cachedSpec)
	agg.specsByAPIServiceName[name] = info
}

// buildMergeSpecLocked creates a new cached mergeSpec from the list of cached specs.
func (agg *specAggregator) buildMergeSpecLocked() cached.Value[*spec.Swagger] {
	apiServices := make([]*apiregistrationv1.APIService, 0, len(agg.specsByAPIServiceName))
	for idx := range agg.specsByAPIServiceName {
		apiServices = append(apiServices, &agg.specsByAPIServiceName[idx].apiService)
	}
	sortByPriority(apiServices)
	caches := make([]cached.Value[*spec.Swagger], len(apiServices))
	for i, apiService := range apiServices {
		caches[i] = &(agg.specsByAPIServiceName[apiService.Name].spec)
	}

	return cached.MergeList(func(results []cached.Result[*spec.Swagger]) (*spec.Swagger, string, error) {
		var merged *spec.Swagger
		eTags := make([]string, 0, len(results))
		for _, specInfo := range results {
			result, eTag, err := specInfo.Get()
			if err != nil {
				// APIService name and err message will be included in the error message as part of decorateError
				klog.Warning(err)
				continue
			}
			if merged == nil {
				merged = &spec.Swagger{}
				*merged = *result
				// Paths, Definitions and parameters are set by
				// MergeSpecsIgnorePathConflictRenamingDefinitionsAndParameters
				merged.Paths = nil
				merged.Definitions = nil
				merged.Parameters = nil
			}
			eTags = append(eTags, eTag)
			if err := aggregator.MergeSpecsIgnorePathConflictRenamingDefinitionsAndParameters(merged, result); err != nil {
				return nil, "", fmt.Errorf("failed to build merge specs: %v", err)
			}
		}
		// Printing the eTags list is stable because it is sorted.
		return merged, fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%#v", eTags)))), nil
	}, caches)
}

// updateServiceLocked updates the spec cache by downloading the latest version of the spec.
func (agg *specAggregator) updateServiceLocked(name string) error {
	specInfo, exists := agg.specsByAPIServiceName[name]
	if !exists {
		return ErrAPIServiceNotFound
	}
	swagger, eTag, err := specInfo.downloader.Get()
	filteredResult := cached.Transform[*spec.Swagger](func(swagger *spec.Swagger, eTag string, err error) (*spec.Swagger, string, error) {
		if err != nil {
			return nil, "", err
		}
		group := specInfo.apiService.Spec.Group
		version := specInfo.apiService.Spec.Version
		return aggregator.FilterSpecByPathsWithoutSideEffects(swagger, []string{"/apis/" + group + "/" + version + "/"}),
			eTag, nil
	}, cached.Result[*spec.Swagger]{Value: swagger, Etag: eTag, Err: err})
	specInfo.spec.Store(filteredResult)
	return err
}

func (agg *specAggregator) AddUpdateAPIService(apiService *apiregistrationv1.APIService, handler http.Handler) error {
	if apiService.Spec.Service == nil {
		return nil
	}
	agg.mutex.Lock()
	defer agg.mutex.Unlock()

	if existingSpec, exists := agg.specsByAPIServiceName[apiService.Name]; !exists {
		specInfo := &openAPISpecInfo{
			apiService: *apiService,
			downloader: NewCacheableDownloader(apiService.Name, agg.downloader, handler),
		}
		specInfo.spec.Store(cached.Result[*spec.Swagger]{Err: fmt.Errorf("spec for api service %s is not yet available",
			apiService.Name)})
		agg.specsByAPIServiceName[apiService.Name] = specInfo
		agg.openAPIVersionedService.UpdateSpecLazy(agg.buildMergeSpecLocked())
	} else {
		existingSpec.apiService = *apiService
		existingSpec.downloader.UpdateHandler(handler)
	}
	return nil
}

// UpdateAPIServiceSpec updates the api service. It is thread safe.
func (agg *specAggregator) UpdateAPIServiceSpec(apiServiceName string) error {
	agg.mutex.Lock()
	defer agg.mutex.Unlock()

	return agg.updateServiceLocked(apiServiceName)
}

// RemoveAPIService removes an api service from OpenAPI aggregation. If it does not exist, no error is returned.
// It is thread safe.
func (agg *specAggregator) RemoveAPIService(apiServiceName string) {
	agg.mutex.Lock()
	defer agg.mutex.Unlock()

	if _, exists := agg.specsByAPIServiceName[apiServiceName]; !exists {
		return
	}
	delete(agg.specsByAPIServiceName, apiServiceName)
	// Re-create the mergeSpec for the new list of api services.
	agg.openAPIVersionedService.UpdateSpecLazy(agg.buildMergeSpecLocked())
}
