package apiserver

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/util/proxy"
	clientGoListersCoreV1 "k8s.io/client-go/listers/core/v1"
	"net/url"
)

// A ServiceResolver  knows how to get a URL given a service.
type ServiceResolver interface {
	ResolveEndpoint(namespace, name string, port int32) (*url.URL, error)
}

type aggregatorEndpointRouting struct {
	services  clientGoListersCoreV1.ServiceLister
	endpoints clientGoListersCoreV1.EndpointsLister
}

// NewEndpointServiceResolver returns a ServiceResolver that chooses one of the service's endpoints.
func NewEndpointServiceResolver(services clientGoListersCoreV1.ServiceLister, endpoints clientGoListersCoreV1.
	EndpointsLister) ServiceResolver {
	return &aggregatorEndpointRouting{services, endpoints}
}

var _ ServiceResolver = &aggregatorEndpointRouting{}

func (r *aggregatorEndpointRouting) ResolveEndpoint(namespace, name string, port int32) (*url.URL, error) {
	return proxy.ResolveEndpoint(r.services, r.endpoints, namespace, name, port)
}

type aggregatorClusterRouting struct {
	services clientGoListersCoreV1.ServiceLister
}

// NewClusterIPServiceResolver returns a ServiceResolver that directly calls the service's cluster IP.
func NewClusterIPServiceResolver(services clientGoListersCoreV1.ServiceLister) ServiceResolver {
	return &aggregatorClusterRouting{services}
}

var _ ServiceResolver = &aggregatorClusterRouting{}

func (r *aggregatorClusterRouting) ResolveEndpoint(namespace, name string, port int32) (*url.URL, error) {
	return proxy.ResolveCluster(r.services, namespace, name, port)
}

type loopbackResolver struct {
	delegate ServiceResolver
	host     *url.URL
}

// NewLoopbackServiceResolver returns a ServiceResolver that routes the kubernetes/default service
// with port 443 to loopback.
func NewLoopbackServiceResolver(delegate ServiceResolver, host *url.URL) ServiceResolver {
	return &loopbackResolver{delegate, host}
}

var _ ServiceResolver = &loopbackResolver{}

func (r *loopbackResolver) ResolveEndpoint(namespace, name string, port int32) (*url.URL, error) {
	if namespace == metav1.NamespaceDefault && name == "kubernetes" && port == 443 {
		return r.host, nil
	}
	return r.delegate.ResolveEndpoint(namespace, name, port)
}
