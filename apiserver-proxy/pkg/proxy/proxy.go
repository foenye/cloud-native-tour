package proxy

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversionscheme "k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	endpointsRequest "k8s.io/apiserver/pkg/endpoints/request"
	clientGoREST "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"net/http"
	"net/http/httputil"
	"net/url"
	runtimeCache "sigs.k8s.io/controller-runtime/pkg/cache"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Proxy struct {
	codecs           serializer.CodecFactory
	scheme           *runtime.Scheme
	restMapper       meta.RESTMapper
	runtimeCache     runtimeCache.Cache
	kubeReverseProxy httputil.ReverseProxy

	jsonSerializer *json.Serializer
	yamlSerializer *json.Serializer
}

func New(restConfig *clientGoREST.Config, codecs serializer.CodecFactory, scheme *runtime.Scheme) (*Proxy, error) {
	transport, err := clientGoREST.TransportFor(restConfig)
	if err != nil {
		return nil, err
	}
	kubeURL, err := url.Parse(restConfig.Host)
	if err != nil {
		return nil, err
	}

	reverseProxy := *httputil.NewSingleHostReverseProxy(kubeURL)
	reverseProxy.Transport = transport
	reverseProxy.ModifyResponse = nil
	reverseProxy.ErrorHandler = nil

	httpClient, err := clientGoREST.HTTPClientFor(restConfig)
	if err != nil {
		return nil, err
	}
	dynamicRESTMapper, err := apiutil.NewDynamicRESTMapper(restConfig, httpClient)
	if err != nil {
		return nil, err
	}
	cache, err := runtimeCache.New(restConfig, runtimeCache.Options{HTTPClient: httpClient, Mapper: dynamicRESTMapper,
		Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return &Proxy{
		codecs:           codecs,
		scheme:           scheme,
		restMapper:       dynamicRESTMapper,
		runtimeCache:     cache,
		kubeReverseProxy: reverseProxy,
		jsonSerializer:   json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{}),
		yamlSerializer:   json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{Yaml: true}),
	}, nil
}

func (proxy *Proxy) Start(ctx context.Context) {
	_ = proxy.runtimeCache.Start(ctx)
}

// Proxy implements http.Handler interface.
var _ http.Handler = &Proxy{}

// ServeHTTP implements http.Handler
func (proxy *Proxy) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	info, _ := endpointsRequest.RequestInfoFrom(ctx)
	switch info.Verb {
	case "get":
		if !info.IsResourceRequest {
			break
		}
		if info.Subresource != "" && info.Subresource != "status" {
			klog.Infof("forward subresource request GET %s to upstream apiserver", info.Path)
			proxy.kubeReverseProxy.ServeHTTP(response, request)
			return
		}

		gvr := schema.GroupVersionResource{Group: info.APIGroup, Version: info.APIVersion, Resource: info.Resource}
		gvk, err := proxy.restMapper.KindFor(gvr)
		if err != nil {
			klog.Errorf("err kind for %v: %s", gvr, gvk)
			break
		}
		klog.V(6).Infof("mapping GroupVersionResource: %v to GroupVersionKind: %v", gvr, gvk)

		var obj runtimeClient.Object
		if runtimeObj, err := proxy.scheme.New(gvk); err != nil {
			objMap := &unstructured.Unstructured{}
			objMap.SetGroupVersionKind(gvk)
			obj = objMap
		} else {
			obj = runtimeObj.(runtimeClient.Object)
		}
		err = proxy.runtimeCache.Get(ctx, types.NamespacedName{Namespace: info.Namespace, Name: info.Name}, obj)
		if err != nil {
			klog.Error("error get object from cache", err)
			break
		}
		klog.V(6).Infof("serve %s from cache", request.URL)
		responsewriters.WriteObjectNegotiated(proxy.codecs, negotiation.DefaultEndpointRestrictions, gvk.GroupVersion(),
			response, request, http.StatusOK, obj, false)
		return
	case "list":
		gvr := schema.GroupVersionResource{Group: info.APIGroup, Version: info.APIVersion, Resource: info.Resource}
		gvk, err := proxy.restMapper.KindFor(gvr)
		if err != nil {
			klog.Errorf("err kind for %v: %s", gvr, gvk)
			break
		}
		klog.V(6).Infof("mapping GroupVersionResource: %v to GroupVersionKind: %v", gvr, gvk)

		options := metav1.ListOptions{}
		if err := metainternalversionscheme.ParameterCodec.DecodeParameters(request.URL.Query(), metav1.
			SchemeGroupVersion, &options); err != nil {
			responsewriters.ErrorNegotiated(errors.NewBadRequest(err.Error()), proxy.codecs, gvk.GroupVersion(),
				response, request)
			return
		}

		var objList unstructured.UnstructuredList
		gvkList := schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind + "List"}
		objList.SetGroupVersionKind(gvkList)

		klog.V(6).Infof("serve %s from cache with options %v", request.URL.Path, options)
		err = proxy.runtimeCache.List(ctx, &objList, &runtimeClient.ListOptions{Namespace: info.Namespace, Raw: &options})
		if err != nil {
			status := responsewriters.ErrorToAPIStatus(err)
			responsewriters.WriteRawJSON(int(status.Code), status, response)
			return
		}

		if _, err := proxy.scheme.New(gvkList); err != nil {
			_, serializerInfo, _ := negotiation.NegotiateOutputMediaType(request, proxy.codecs, negotiation.DefaultEndpointRestrictions)
			switch serializerInfo.MediaType {
			case "":
				fallthrough
			case "application/json":
				response.Header().Add("Content-Type", "application/json")
				_ = proxy.jsonSerializer.Encode(&objList, response)
			case "application/yaml":
				response.Header().Add("Content-Type", "application/yaml")
				_ = proxy.yamlSerializer.Encode(&objList, response)
			default:
				status := metav1.Status{
					Status:  metav1.StatusFailure,
					Code:    http.StatusNotAcceptable,
					Reason:  metav1.StatusReasonNotAcceptable,
					Message: "only the following media types are accepted: application/json, application/yaml",
				}
				responsewriters.WriteRawJSON(int(status.Code), status, response)
			}
			return
		}
		responsewriters.WriteObjectNegotiated(proxy.codecs, negotiation.DefaultEndpointRestrictions, gvk.GroupVersion(),
			response, request, http.StatusOK, &objList, false)
		return
	default:
	}
	proxy.kubeReverseProxy.ServeHTTP(response, request)
}
