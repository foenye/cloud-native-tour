package apiserver

import (
	apiregistrationv1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1helper "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1/helper"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	endpointsMetrics "k8s.io/apiserver/pkg/endpoints/metrics"
	endpointsRequest "k8s.io/apiserver/pkg/endpoints/request"
	apiserverFeatures "k8s.io/apiserver/pkg/features"
	apiserverUtilFeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/apiserver/pkg/util/flowcontrol"
	apiserverUtilProxy "k8s.io/apiserver/pkg/util/proxy"
	"k8s.io/apiserver/pkg/util/x509metrics"
	"k8s.io/client-go/transport"
	"k8s.io/component-base/tracing"
	"k8s.io/klog/v2"
	"net/http"
	"net/url"
	"sync/atomic"
)

const (
	aggregatorComponent string = "aggregator"
)

type certKeyFunc func() ([]byte, []byte)

type proxyHandlingInfo struct {
	// local indicates that this APIServices is locally satisfied
	local bool

	// name is the name of the APIService
	name string
	// transportConfig holds the information for building a round tripper.
	transportConfig *transport.Config
	// transportBuildingError is an error produced which building the transport.  If this is non-nil, it will be reported
	// to clients.
	transportBuildingError error
	// proxyRoundTripper is the re-use-able portion of the transport.  It does not vary with any request.
	proxyRoundTripper http.RoundTripper
	// serviceName is the name of the service this handler proxies to
	serviceName string
	// serviceNamespace is the namespace the service lives in
	serviceNamespace string
	// serviceAvailable indicates this APIService is available or not
	serviceAvailable bool
	// servicePort is the port of the service this handler proxies to
	servicePort int32
}

var _ proxy.ErrorResponder = &responder{}

// responder implements proxy.ErrorResponder for assisting a connector in writing objects or errors.
type responder struct {
	response http.ResponseWriter
}

func (res *responder) Error(_ http.ResponseWriter, _ *http.Request, err error) {
	http.Error(res.response, err.Error(), http.StatusServiceUnavailable)
}

func (res *responder) Object(statusCode int, obj runtime.Object) {
	responsewriters.WriteRawJSON(statusCode, obj, res.response)
}

type proxyHandlerImplementation interface {
	http.Handler

	setServiceAvailable()
	updateAPIService(apiService *apiregistrationv1.APIService)
}

var _ proxyHandlerImplementation = &proxyHandler{}

// proxyHandler provides a http.Handler which will proxy traffic to locations specified by items implementing Redirector.
type proxyHandler struct {
	// localDelegate is used to satisfy local APIServices
	localDelegate http.Handler

	// proxyCurrentCertKeyContent holds the client cert used to identify this proxy.
	// Backing APIServices use this to confirm the proxy's identity.
	proxyCurrentCertKeyContent certKeyFunc
	proxyTransportDial         *transport.DialHolder

	// Endpoints based routing to map from cluster IP to routable IP.
	serviceResolver ServiceResolver

	handlingInfo atomic.Value

	// reject to froward redirect response
	rejectForwardingRedirects bool

	// tracerProvider is used to warp the proxy transport and handler with tracing
	tracerProvider tracing.TracerProvider
}

func (handler *proxyHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	value := handler.handlingInfo.Load()
	if value == nil {
		handler.localDelegate.ServeHTTP(response, request)
		return
	}
	handingInfo := value.(proxyHandlingInfo)
	if handingInfo.local {
		if handler.localDelegate == nil {
			http.Error(response, "", http.StatusNotFound)
			return
		}
		handler.localDelegate.ServeHTTP(response, request)
		return
	}

	if !handingInfo.serviceAvailable {
		proxyError(response, request, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	if handingInfo.transportBuildingError != nil {
		proxyError(response, request, handingInfo.transportBuildingError.Error(), http.StatusInternalServerError)
		return
	}

	userinfo, exists := endpointsRequest.UserFrom(request.Context())
	if !exists {
		proxyError(response, request, "missing user", http.StatusInternalServerError)
		return
	}

	// write a new location based on the existing request pointed at the target service
	location := &url.URL{}
	location.Scheme = "https"
	endpoint, err := handler.serviceResolver.ResolveEndpoint(handingInfo.serviceNamespace, handingInfo.serviceName,
		handingInfo.servicePort)
	if err != nil {
		klog.Errorf("error resoling %s/%s: %v", handingInfo.serviceNamespace, handingInfo.serviceName, err)
		proxyError(response, request, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	location.Host = endpoint.Host
	location.Path = request.URL.Path
	location.RawQuery = request.URL.Query().Encode()
	proxiedRequest, cancelFn := apiserverUtilProxy.NewRequestForProxy(location, request)
	defer cancelFn()

	if handingInfo.proxyRoundTripper == nil {
		proxyError(response, request, "", http.StatusNotFound)
		return
	}

	proxyRoundTripper := handingInfo.proxyRoundTripper
	requestUpgrade := httpstream.IsUpgradeRequest(request)

	var userUID string
	if apiserverUtilFeature.DefaultFeatureGate.Enabled(apiserverFeatures.RemoteRequestHeaderUID) {
		userUID = userinfo.GetUID()
	}

	proxyRoundTripper = transport.NewAuthProxyRoundTripper(userinfo.GetName(), userUID, userinfo.GetGroups(),
		userinfo.GetExtra(), proxyRoundTripper)

	if apiserverUtilFeature.DefaultFeatureGate.Enabled(apiserverFeatures.APIServerTracing) && !requestUpgrade {
		tracingWrapper := tracing.WrapperFor(handler.tracerProvider)
		proxyRoundTripper = tracingWrapper(proxyRoundTripper)
	}

	// If we are upgrading, then the upgrade path tries to use this request with TLS config we provide, but it does
	// NOT use the proxyRoundTripper. It's a direct dial bypasses the proxyRoundTripper. This means that we hava to
	// attach the "correct" user headers to the request ahead of time.
	if requestUpgrade {
		transport.SetAuthProxyHeaders(proxiedRequest, userinfo.GetName(), userUID, userinfo.GetGroups(), userinfo.GetExtra())
	}

	upgradedHandler := proxy.NewUpgradeAwareHandler(location, proxyRoundTripper, true, requestUpgrade, &responder{response: response})
	if handler.rejectForwardingRedirects {
		upgradedHandler.RejectForwardingRedirects = true
	}
	flowcontrol.RequestDelegated(request.Context())
	upgradedHandler.ServeHTTP(response, proxiedRequest)
}

// setServiceAvailable sets serviceAvailable value on proxyHandler, not thread safe
func (handler *proxyHandler) setServiceAvailable() {
	handingInfo := handler.handlingInfo.Load().(proxyHandlingInfo)
	handingInfo.serviceAvailable = true
	handler.handlingInfo.Store(handingInfo)
}

func (handler *proxyHandler) updateAPIService(apiService *apiregistrationv1.APIService) {
	if apiService.Spec.Service == nil {
		handler.handlingInfo.Store(proxyHandlingInfo{local: true})
		return
	}

	proxyClientCert, proxyClientKey := handler.proxyCurrentCertKeyContent()

	transportConfig := &transport.Config{
		TLS: transport.TLSConfig{
			Insecure:   apiService.Spec.InsecureSkipTLSVerify,
			ServerName: apiService.Spec.Service.Name + "." + apiService.Spec.Service.Namespace + ".svc",
			CertData:   proxyClientCert,
			KeyData:    proxyClientKey,
			CAData:     apiService.Spec.CABundle,
		},
		DialHolder: handler.proxyTransportDial,
	}
	transportConfig.Wrap(x509metrics.NewDeprecatedCertificateRoundTripperWrapperConstructor(
		x509MissingSANCounter,
		x509InsecureSHA1Counter,
	))

	handingInfo := proxyHandlingInfo{
		name:             apiService.Name,
		transportConfig:  transportConfig,
		serviceName:      apiService.Spec.Service.Name,
		serviceNamespace: apiService.Spec.Service.Namespace,
		servicePort:      *apiService.Spec.Service.Port,
		serviceAvailable: apiregistrationv1helper.IsAPIServiceConditionTrue(apiService, apiregistrationv1.Available),
	}
	handingInfo.proxyRoundTripper, handingInfo.transportBuildingError = transport.New(handingInfo.transportConfig)
	if handingInfo.transportBuildingError != nil {
		klog.Warning(handingInfo.transportBuildingError.Error())
	}
	handler.handlingInfo.Store(handingInfo)
}

func proxyError(response http.ResponseWriter, request *http.Request, error string, code int) {
	http.Error(response, error, code)

	ctx := request.Context()
	info, exists := endpointsRequest.RequestInfoFrom(ctx)
	if !exists {
		klog.Warning("no RequestInfo found in the context")
		return
	}
	// TODO: record long-running request differently? The long-running check func does not necessarily match the one of the aggregated apiserver
	endpointsMetrics.RecordRequestTermination(request, info, aggregatorComponent, code)
}
