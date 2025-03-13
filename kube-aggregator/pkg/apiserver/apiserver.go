package apiserver

import (
	"context"
	"fmt"
	apiregistrationv1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1helper "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1/helper"
	apiregistrationv1beta1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	aggregatorAPIServerScheme "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apiserver/scheme"
	generatedClientset "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/clientset_generated/clientset"
	generatedClientInformers "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/informers/externalversions"
	generatedClientListersV1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/listers/apiregistration/v1"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/controllers/openapi"
	openapiV2Aggregator "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/controllers/openapi/aggregator"
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/controllers/openapiv3"
	openapiV3Aggregator "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/controllers/openapiv3/aggregator"
	controllerStatusLocal "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/controllers/status/local"
	controllerStatusMetrics "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/controllers/status/metrics"
	controllerStatusRemote "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/controllers/status/remote"
	registryAPIServiceREST "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/registry/apiservice/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/endpoints/discovery/aggregated"
	"k8s.io/apiserver/pkg/features"
	peerReconcilers "k8s.io/apiserver/pkg/reconcilers"
	genericAPIServer "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"
	"k8s.io/apiserver/pkg/server/egressselector"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/transport"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/component-base/tracing"
	kubeOpenAPICommon "k8s.io/kube-openapi/pkg/common"
	"net/http"
	"strings"
	"sync"
	"time"
)

// making sure we only register metrics once into legacy registry.
var registerIntoLegacyRegistryOnce sync.Once

func init() {
	// we need to add the options (like ListOptions) to empty v1
	metav1.AddToGroupVersion(aggregatorAPIServerScheme.Scheme, schema.GroupVersion{Group: "", Version: "v1"})
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	aggregatorAPIServerScheme.Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

const (
	// legacyAPIServiceName is the fixed name of the only non-groupified API version.
	legacyAPIServiceName = "v1"
	// StorageVersionPostStartHookName is the name of the storage version updater post start hook.
	StorageVersionPostStartHookName = "built-in-resources-storage-version-updater"
)

// ExtraConfig represents APIServices-specific configuration.
type ExtraConfig struct {
	// PeerAdvertiseAddress is the IP for this kube-apiserver which is used by peer apiservers to route a request to
	// this apiserver. This happens in cases where the peer is not able to serve the request due to version skew. If
	// unset, AdvertiseAddress/BindAddress will be used.
	PeerAdvertiseAddress peerReconcilers.PeerAdvertiseAddress
	// ProxyClientCert/Key are the client cert used to identity this proxy. Backing APIServices use this to confirm
	// the proxy's identity.
	ProxyClientCertFile string
	ProxyClientKeyFile  string
	// If present, the Dial method will be used for dialing out to delegate apiservices.
	ProxyTransport *http.Transport
	// Mechanism by which the Aggregator will resolve services. Required.
	ServiceResolver           ServiceResolver
	RejectForwardingRedirects bool
	// DisableRemoteAvailableConditionController disables the controller that updates the Available conditions for
	// remote APIServices via querying endpoints of the referenced services. In generic control-plane use-cases,
	// the concept of services and endpoints might differ, and might require another implementation of this controller.
	// Local APIService are reconciled nevertheless.
	DisableRemoteAvailableConditionController bool
}

type ConfigImplementation interface {
	Complete() CompletedConfig
}

var _ ConfigImplementation = &Config{}

// Config represents the configuration needed tro create an APIAggregator.
type Config struct {
	GenericConfig *genericAPIServer.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// Complete fills in any fields not set that are required to hava valid data. It's mutating the receiver.
func (config *Config) Complete() CompletedConfig {
	configCompleted := completedConfig{config.GenericConfig.Complete(), &config.ExtraConfig}

	// The kube aggregator wires its own discovery mechanism.
	// TODO: eventually collapse this by extracting all of the discovery out.
	config.GenericConfig.EnableDiscovery = false

	return CompletedConfig{&configCompleted}
}

type completedConfigImplementation interface {
	NewWithDelegate(delegationTarget genericAPIServer.DelegationTarget) (*APIAggregator, error)
}

var _ completedConfigImplementation = completedConfig{}

type completedConfig struct {
	GenericConfig genericAPIServer.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// NewWithDelegate returns a new instance of APIAggregator from given config.
func (cfg completedConfig) NewWithDelegate(delegationTarget genericAPIServer.DelegationTarget) (*APIAggregator, error) {
	apiServer, err := cfg.GenericConfig.New("kube-aggregator", delegationTarget)
	if err != nil {
		return nil, err
	}

	aggregationClient, err := generatedClientset.NewForConfig(cfg.GenericConfig.LoopbackClientConfig)
	if err != nil {
		return nil, err
	}
	informerFactory := generatedClientInformers.NewSharedInformerFactory(aggregationClient,
		// this is effectively used as a refresh interval right now.  Might want to do something nicer later on.
		5*time.Minute)

	// apiServiceRegistrationControllerInitiated is closed when APIServiceRegistrationController has finished "installing" all known APIServices.
	// At this point we know that the proxy handler knows about APIServices and can handle client requests.
	// Before it might have resulted in a 404 response which could have serious consequences for some controllers like  GC and NS
	//
	// Note that the APIServiceRegistrationController waits for APIServiceInformer to synced before doing its work.
	apiServiceRegistrationControllerInitiated := make(chan struct{})
	if err := apiServer.RegisterMuxAndDiscoveryCompleteSignal("apiServiceRegistrationControllerInitiated",
		apiServiceRegistrationControllerInitiated); err != nil {
		return nil, err
	}

	var proxyTransportDial *transport.DialHolder
	if cfg.GenericConfig.EgressSelector != nil {
		egressDialer, err := cfg.GenericConfig.EgressSelector.Lookup(egressselector.Cluster.AsNetworkContext())
		if err != nil {
			return nil, err
		}
		if egressDialer != nil {
			proxyTransportDial = &transport.DialHolder{Dial: egressDialer}
		}
	} else if cfg.ExtraConfig.ProxyTransport != nil && cfg.ExtraConfig.ProxyTransport.DialContext != nil {
		proxyTransportDial = &transport.DialHolder{Dial: cfg.ExtraConfig.ProxyTransport.DialContext}
	}
	apiAggregator := &APIAggregator{
		GenericAPIServer:           apiServer,
		delegateHandler:            delegationTarget.UnprotectedHandler(),
		proxyTransportDial:         proxyTransportDial,
		proxyHandlers:              map[string]*proxyHandler{},
		handledGroupVersions:       map[string]sets.Set[string]{},
		lister:                     informerFactory.Apiregistration().V1().APIServices().Lister(),
		APIRegistrationInformers:   informerFactory,
		serviceResolver:            cfg.ExtraConfig.ServiceResolver,
		openAPIConfig:              cfg.GenericConfig.OpenAPIConfig,
		openAPIV3Config:            cfg.GenericConfig.OpenAPIV3Config,
		proxyCurrentCertKeyContent: func() (certBytes []byte, keyBytes []byte) { return nil, nil },
		rejectForwardRedirects:     cfg.ExtraConfig.RejectForwardingRedirects,
		tracerProvider:             cfg.GenericConfig.TracerProvider,
	}
	apiGroupInfo := registryAPIServiceREST.NewRESTStorage(cfg.GenericConfig.MergedResourceConfig, cfg.GenericConfig.
		RESTOptionsGetter, false)
	if err := apiAggregator.GenericAPIServer.InstallAPIGroups(&apiGroupInfo); err != nil {
		return nil, err
	}

	enabledVersions := sets.New[string]()
	for version := range apiGroupInfo.VersionedResourcesStorageMap {
		enabledVersions.Insert(version)
	}
	if !enabledVersions.Has(apiregistrationv1.SchemeGroupVersion.Version) {
		return nil, fmt.Errorf("API group/version %s must be enabled", apiregistrationv1.SchemeGroupVersion.String())
	}

	apisHandler := &apisHandler{
		codecs:         aggregatorAPIServerScheme.Codecs,
		lister:         apiAggregator.lister,
		discoveryGroup: discoveryGroup(enabledVersions),
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.AggregatedDiscoveryEndpoint) {
		apisHandlerWithAggregationSupport := aggregated.WrapAggregatedDiscoveryToHandler(apisHandler, apiAggregator.
			GenericAPIServer.AggregatedDiscoveryGroupManager)
		apiAggregator.GenericAPIServer.Handler.NonGoRestfulMux.Handle("/apis", apisHandlerWithAggregationSupport)
	} else {
		apiAggregator.GenericAPIServer.Handler.NonGoRestfulMux.Handle("/apis", apisHandler)
	}
	apiAggregator.GenericAPIServer.Handler.NonGoRestfulMux.UnlistedHandle("/apis/", apisHandler)

	apiServiceRegistrationController := NewAPIServiceRegistrationController(informerFactory.Apiregistration().V1().APIServices(), apiAggregator)
	if len(cfg.ExtraConfig.ProxyClientCertFile) > 0 && len(cfg.ExtraConfig.ProxyClientKeyFile) > 0 {
		aggregatorProxyCerts, err := dynamiccertificates.NewDynamicServingContentFromFiles("aggregator-proxy-cert",
			cfg.ExtraConfig.ProxyClientCertFile,
			cfg.ExtraConfig.ProxyClientKeyFile)
		if err != nil {
			return nil, err
		}
		// We are passing the context to ProxyCerts.RunOnce as it needs to implement RunOnce(ctx) however the
		// context is not used at all. So passing a empty context shouldn't be a problem
		if err := aggregatorProxyCerts.RunOnce(context.Background()); err != nil {
			return nil, err
		}
		aggregatorProxyCerts.AddListener(apiServiceRegistrationController)
		apiAggregator.proxyCurrentCertKeyContent = aggregatorProxyCerts.CurrentCertKeyContent

		apiAggregator.GenericAPIServer.AddPostStartHookOrDie("aggregator-reload-proxy-client-cert",
			func(postStartHookContext genericAPIServer.PostStartHookContext) error {
				go aggregatorProxyCerts.Run(postStartHookContext, 1)
				return nil
			},
		)
	}

	apiAggregator.GenericAPIServer.AddPostStartHookOrDie("start-kube-aggregator-informers",
		func(context genericAPIServer.PostStartHookContext) error {
			informerFactory.Start(context.Done())
			cfg.GenericConfig.SharedInformerFactory.Start(context.Done())
			return nil
		},
	)

	// create shared (remote and local) availability metrics
	// TODO: decouple from legacyregistry
	metrics := controllerStatusMetrics.New()
	registerIntoLegacyRegistryOnce.Do(func() { err = metrics.Register(legacyregistry.Register, legacyregistry.CustomRegister) })
	if err != nil {
		return nil, err
	}

	// always run local availability controller
	local, err := controllerStatusLocal.New(informerFactory.Apiregistration().V1().APIServices(),
		aggregationClient.ApiregistrationV1(),
		metrics)
	if err != nil {
		return nil, err
	}
	apiAggregator.GenericAPIServer.AddPostStartHookOrDie("apiservice-status-local-available-controller",
		func(context genericAPIServer.PostStartHookContext) error {
			// if we end up blocking for long periods of time, we may need to increase workers.
			go local.Run(5, context.Done())
			return nil
		})

	// conditionally run remote availability controller. This could be replaced in certain
	// generic controlplane use-cases where there is another concept of services and/or endpoints.
	if !cfg.ExtraConfig.DisableRemoteAvailableConditionController {
		remote, err := controllerStatusRemote.New(
			informerFactory.Apiregistration().V1().APIServices(),
			cfg.GenericConfig.SharedInformerFactory.Core().V1().Services(),
			cfg.GenericConfig.SharedInformerFactory.Core().V1().Endpoints(),
			aggregationClient.ApiregistrationV1(),
			proxyTransportDial,
			(func() ([]byte, []byte))(apiAggregator.proxyCurrentCertKeyContent),
			apiAggregator.serviceResolver,
			metrics,
		)
		if err != nil {
			return nil, err
		}
		apiAggregator.GenericAPIServer.AddPostStartHookOrDie("apiservice-status-remote-available-controller",
			func(context genericAPIServer.PostStartHookContext) error {
				// if we end up blocking for long periods of time, we may need to increase workers.
				go remote.Run(5, context.Done())
				return nil
			})
	}

	apiAggregator.GenericAPIServer.AddPostStartHookOrDie("apiservice-registration-controller",
		func(context genericAPIServer.PostStartHookContext) error {
			go apiServiceRegistrationController.Run(context.Done(), apiServiceRegistrationControllerInitiated)
			select {
			case <-context.Done():
			case <-apiServiceRegistrationControllerInitiated:
			}
			return nil
		})

	if utilfeature.DefaultFeatureGate.Enabled(features.AggregatedDiscoveryEndpoint) {
		apiAggregator.discoveryAggregationController = NewDiscoveryManager(
			// Use aggregator as the source name to avoid overwriting native/CRD
			// groups
			apiAggregator.GenericAPIServer.AggregatedDiscoveryGroupManager.WithSource(aggregated.AggregatorSource),
		)
		// Setup discovery endpoint
		apiAggregator.GenericAPIServer.AddPostStartHookOrDie("apiservice-discovery-controller", func(context genericAPIServer.PostStartHookContext) error {
			// Discovery aggregation depends on the apiservice registration controller
			// having the full list of APIServices already synced
			select {
			case <-context.Done():
				return nil
			// Context cancelled, should abort/clean goroutines
			case <-apiServiceRegistrationControllerInitiated:
			}

			// Run discovery manager's worker to watch for new/removed/updated
			// APIServices to the discovery document can be updated at runtime
			// When discovery is ready, all APIServices will be present, with APIServices
			// that have not successfully synced discovery to be present but marked as Stale.
			discoverySyncedCh := make(chan struct{})
			go apiAggregator.discoveryAggregationController.Run(context.Done(), discoverySyncedCh)

			select {
			case <-context.Done():
				return nil
			// Context cancelled, should abort/clean goroutines
			case <-discoverySyncedCh:
				// API services successfully sync
			}
			return nil
		})
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.StorageVersionAPI) && utilfeature.DefaultFeatureGate.Enabled(
		features.APIServerIdentity) {
		// Spawn a goroutine in aggregator apiserver to update storage version for
		// all built-in resources
		apiAggregator.GenericAPIServer.AddPostStartHookOrDie(StorageVersionPostStartHookName, func(hookContext genericAPIServer.PostStartHookContext) error {
			kubeClient, err := kubernetes.NewForConfig(hookContext.LoopbackClientConfig)
			if err != nil {
				return err
			}
			if err := wait.PollUntilContextCancel(wait.ContextForChannel(hookContext.Done()), 100*time.Millisecond, true,
				func(ctx context.Context) (done bool, err error) {
					_, err = kubeClient.CoordinationV1().Leases(metav1.NamespaceSystem).Get(context.TODO(), apiAggregator.
						GenericAPIServer.APIServerID, metav1.GetOptions{})
					if err != nil {
						return false, err
					}
					return true, nil
				}); err != nil {
				return fmt.Errorf("failed to wait for apiserver-identity lease %s to be created: %v",
					apiAggregator.GenericAPIServer.APIServerID, err)
			}
			// Technically an apiserver only needs to update storage version once during bootstrap.
			// Reconcile StorageVersion objects every 10 minutes will help in the case that the
			// StorageVersion objects get accidentally modified/deleted by a different agent. In that
			// case, the reconciliation ensures future storage migration still works. If nothing gets
			// changed, the reconciliation update is a noop and gets short-circuited by the apiserver,
			// therefore won't change the resource version and trigger storage migration.
			go func() {
				_ = wait.PollUntilContextCancel(wait.ContextForChannel(hookContext.Done()), 10*time.Minute, true,
					func(ctx context.Context) (done bool, err error) {
						// All apiservers (aggregator-apiserver, kube-apiserver, apiextensions-apiserver)
						// share the same generic apiserver config. The same StorageVersion manager is used
						// to register all built-in resources when the generic apiservers install APIs.
						apiAggregator.GenericAPIServer.StorageVersionManager.UpdateStorageVersions(hookContext.
							LoopbackClientConfig, apiAggregator.GenericAPIServer.APIServerID)
						return false, nil
					})
			}()
			// Once the storage version updater finishes the first round of update,
			// the PostStartHook will return to unblock /healthz. The handler chain
			// won't block write requests anymore. Check every second since it's not
			// expensive.
			_ = wait.PollUntilContextCancel(wait.ContextForChannel(hookContext.Done()), 1*time.Second, true,
				func(ctx context.Context) (done bool, err error) {
					return apiAggregator.GenericAPIServer.StorageVersionManager.Completed(), nil
				})
			return nil
		})
	}
	return apiAggregator, nil
}

// CompletedConfig same as Config, just to swap private object.
type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

type runnable interface {
	RunWithContext(ctx context.Context) error
}

type APIAggregatorImplementation interface {
	AddAPIService(apiService *apiregistrationv1.APIService) error
	RemoveAPIService(apiServiceName string)
	PreparedRun() (PreparedAPIAggregatorImplementation, error)
}

var _ APIAggregatorImplementation = &APIAggregator{}

// APIAggregator contains state for a Kubernetes cluster master/api server.
type APIAggregator struct {
	GenericAPIServer *genericAPIServer.GenericAPIServer

	// provided for easier embedding
	APIRegistrationInformers generatedClientInformers.SharedInformerFactory

	delegateHandler http.Handler

	// proxyCurrentCertKeyContent holds the client cert used to identity this proxy.
	// Backing APIServices use this to confirm the proxy's identity
	proxyCurrentCertKeyContent certKeyFunc
	proxyTransportDial         *transport.DialHolder

	// proxyHandlers are the proxy handlers that are currently registered, keyed by APIService.name
	proxyHandlers map[string]*proxyHandler
	// handledGroupVersions contains the groups that already have routes. The key is the name of the group and the
	// value is the versions for the group.
	handledGroupVersions map[string]sets.Set[string]

	// lister is used to add group handling for /apis/<group> aggregator lookups based on controller state.
	lister generatedClientListersV1.APIServiceLister

	// Information needed to determine routing for the aggregator.
	serviceResolver ServiceResolver

	// Enable swagger and/or  OpenAPI if these configs are non-nil.
	openAPIConfig *kubeOpenAPICommon.Config

	// Enable OpenAPI v3 if these configs are non-nil.
	openAPIV3Config *kubeOpenAPICommon.OpenAPIV3Config

	// openAPIAggregationController downloads and merges OpenAPI v2 specs.
	openAPIAggregationController *openapi.AggregationController

	// openAPIV3AggregationController downloads and caches OpenAPI v3 specs.
	openAPIV3AggregationController *openapiv3.AggregationController

	// discoveryAggregationController downloads and caches discovery documents from all aggregated api services so
	// they are available from /apis endpoint when discovery with resources are requested.
	discoveryAggregationController DiscoveryAggregationController

	// rejectForwardRedirects is whether to allow to forward redirect response.
	rejectForwardRedirects bool

	// tracerProvider is used to warp the proxy transport and handler with tracing.
	tracerProvider tracing.TracerProvider
}

// AddAPIService adds an API service.  It is not thread-safe, so only call it on one thread at a time please.
// It's a slow moving API, so it's ok to run the controller on a single thread.
func (aggregator *APIAggregator) AddAPIService(apiService *apiregistrationv1.APIService) error {
	// If the proxyHandler already exists, it needs to be updated. The aggregation bits do not sine they are
	//	wired against listers because they require multiple resources to respond.
	if proxyHandler, exists := aggregator.proxyHandlers[apiService.Name]; exists {
		proxyHandler.updateAPIService(apiService)
		if aggregator.openAPIAggregationController != nil {
			aggregator.openAPIAggregationController.UpdateAPIService(proxyHandler, apiService)
		}
		if aggregator.openAPIV3AggregationController != nil {
			aggregator.openAPIV3AggregationController.UpdateAPIService(proxyHandler, apiService)
		}

		// Forward calls to discovery manager to update discovery document
		if aggregator.discoveryAggregationController != nil {
			handlerCopied := *proxyHandler
			handlerCopied.setServiceAvailable()
			aggregator.discoveryAggregationController.AddAPIService(apiService, &handlerCopied)
		}
		return nil
	}

	proxyPath := strings.Join([]string{"/apis", apiService.Spec.Group, apiService.Spec.Version}, "/")
	// v1. is a special case for the legacy API.  It proxies to a wider set of endpoints.
	if apiService.Name == legacyAPIServiceName {
		proxyPath = "/api"
	}

	// register the proxy handler
	proxyHandler := &proxyHandler{
		localDelegate:              aggregator.delegateHandler,
		proxyCurrentCertKeyContent: aggregator.proxyCurrentCertKeyContent,
		proxyTransportDial:         aggregator.proxyTransportDial,
		serviceResolver:            aggregator.serviceResolver,
		rejectForwardingRedirects:  aggregator.rejectForwardRedirects,
		tracerProvider:             aggregator.tracerProvider,
	}
	proxyHandler.updateAPIService(apiService)
	if aggregator.openAPIAggregationController != nil {
		aggregator.openAPIAggregationController.AddAPIService(proxyHandler, apiService)
	}
	if aggregator.openAPIV3AggregationController != nil {
		aggregator.openAPIV3AggregationController.AddAPIService(proxyHandler, apiService)
	}
	if aggregator.discoveryAggregationController != nil {
		aggregator.discoveryAggregationController.AddAPIService(apiService, proxyHandler)
	}

	aggregator.proxyHandlers[apiService.Name] = proxyHandler
	aggregator.GenericAPIServer.Handler.NonGoRestfulMux.Handle(proxyPath, proxyHandler)
	aggregator.GenericAPIServer.Handler.NonGoRestfulMux.UnlistedHandlePrefix(proxyPath+"/", proxyHandler)

	// If we're dealing with the legacy group, we're done here.
	if apiService.Name == legacyAPIServiceName {
		return nil
	}

	// If we've already registered the path with the handler, we don't want to do it again.
	versions, exists := aggregator.handledGroupVersions[apiService.Spec.Group]
	if exists {
		versions.Insert(apiService.Spec.Version)
		return nil
	}

	// It's time to register the group aggregation endpoint.
	groupPath := "/apis/" + apiService.Spec.Group
	groupDiscoveryHandler := &apiGroupHandler{
		codecs:    aggregatorAPIServerScheme.Codecs,
		groupName: apiService.Spec.Group,
		lister:    aggregator.lister,
		delegate:  aggregator.delegateHandler,
	}
	// aggregation is protected
	aggregator.GenericAPIServer.Handler.NonGoRestfulMux.Handle(groupPath, groupDiscoveryHandler)
	aggregator.GenericAPIServer.Handler.NonGoRestfulMux.UnlistedHandlePrefix(groupPath+"/", groupDiscoveryHandler)
	aggregator.handledGroupVersions[apiService.Spec.Group] = sets.New[string](apiService.Spec.Version)
	return nil
}

// RemoveAPIService removes the APIService from being handled.  It's not thread-safe, so only call it on one
// thread at a time please.  It's a slow moving API, so it's ok to run the controller on a single thread.
func (aggregator *APIAggregator) RemoveAPIService(apiServiceName string) {
	// Forward calls to discovery manager to update discovery document.
	if aggregator.discoveryAggregationController != nil {
		aggregator.discoveryAggregationController.RemoveAPIService(apiServiceName)
	}

	groupVersion := apiregistrationv1helper.APIServiceNameToGroupVersion(apiServiceName)

	proxyPath := strings.Join([]string{"/apis", groupVersion.Group, groupVersion.Version}, "/")
	// v1. is a special case for the legacy API.  It proxies to a wider set of endpoints.
	if apiServiceName == legacyAPIServiceName {
		proxyPath = "/api"
	}
	aggregator.GenericAPIServer.Handler.NonGoRestfulMux.Unregister(proxyPath)
	aggregator.GenericAPIServer.Handler.NonGoRestfulMux.Unregister(proxyPath + "/")
	if aggregator.openAPIV3AggregationController != nil {
		aggregator.openAPIV3AggregationController.RemoveAPIService(apiServiceName)
	}
	if aggregator.discoveryAggregationController != nil {
		aggregator.discoveryAggregationController.RemoveAPIService(apiServiceName)
	}
	delete(aggregator.proxyHandlers, apiServiceName)

	versions, exists := aggregator.handledGroupVersions[groupVersion.Version]
	if !exists {
		return
	}
	versions.Delete(groupVersion.Version)
	if versions.Len() > 0 {
		return
	}
	delete(aggregator.handledGroupVersions, groupVersion.Version)
	groupPath := "/apis/" + groupVersion.Group
	aggregator.GenericAPIServer.Handler.NonGoRestfulMux.Unregister(groupPath)
	aggregator.GenericAPIServer.Handler.NonGoRestfulMux.Unregister(groupPath + "/")
}

func (aggregator *APIAggregator) PreparedRun() (PreparedAPIAggregatorImplementation, error) {
	// Add post start hook before generic PrepareRun in order to be before /healthz installation.
	if aggregator.openAPIConfig != nil {
		aggregator.GenericAPIServer.AddPostStartHookOrDie("apiservice-openapi-controller", func(context genericAPIServer.PostStartHookContext) error {
			go aggregator.openAPIAggregationController.Run(context.Done())
			return nil
		})
	}
	if aggregator.openAPIV3Config != nil {
		aggregator.GenericAPIServer.AddPostStartHookOrDie("apiservice-openapiv3-controller", func(context genericAPIServer.PostStartHookContext) error {
			go aggregator.openAPIV3AggregationController.Run(context.Done())
			return nil
		})
	}

	prepared := aggregator.GenericAPIServer.PrepareRun()

	// Delay OpenAPI setup until the delegate had a chance to set up their OpenAPI handlers.
	if aggregator.openAPIConfig != nil {
		specDownloader := openapiV2Aggregator.NewDownloader()
		registeredAggregator, err := openapiV2Aggregator.BuildAndRegisterAggregator(
			&specDownloader,
			aggregator.GenericAPIServer.NextDelegate(),
			aggregator.GenericAPIServer.Handler.GoRestfulContainer.RegisteredWebServices(),
			aggregator.openAPIConfig,
			aggregator.GenericAPIServer.Handler.NonGoRestfulMux)
		if err != nil {
			return preparedAPIAggregator{}, err
		}
		aggregator.openAPIAggregationController = openapi.NewAggregationController(&specDownloader, registeredAggregator)
	}
	if aggregator.openAPIV3Config != nil {
		specDownloader := openapiV3Aggregator.NewDownloader()
		registeredAggregator, err := openapiV3Aggregator.BuildAndRegisterAggregator(
			specDownloader,
			aggregator.GenericAPIServer.NextDelegate(),
			aggregator.GenericAPIServer.Handler.GoRestfulContainer,
			aggregator.openAPIV3Config,
			aggregator.GenericAPIServer.Handler.NonGoRestfulMux)
		if err != nil {
			return preparedAPIAggregator{}, err
		}
		aggregator.openAPIV3AggregationController = openapiv3.NewAggregationController(registeredAggregator)
	}

	return preparedAPIAggregator{APIAggregator: aggregator, runnable: prepared}, nil
}

type PreparedAPIAggregatorImplementation interface {
	Run(ctx context.Context) error
}

var _ PreparedAPIAggregatorImplementation = &preparedAPIAggregator{}

// preparedAPIAggregator is a generic API server private wrapper that enforces a call of PrepareRun() before Run can
// be invoked.
type preparedAPIAggregator struct {
	*APIAggregator
	runnable runnable
}

func (prepared preparedAPIAggregator) Run(ctx context.Context) error {
	return prepared.runnable.RunWithContext(ctx)
}

// DefaultAPIResourceConfigSource returns default configuration for an APIResource.
func DefaultAPIResourceConfigSource() *serverstorage.ResourceConfig {
	resourceConfig := serverstorage.NewResourceConfig()
	// NOTE: GroupVersions listed here will be enabled by default. Don't put alpha versions in the list.
	resourceConfig.EnableVersions(
		apiregistrationv1.SchemeGroupVersion,
		apiregistrationv1beta1.SchemeGroupVersion,
	)

	return resourceConfig
}
