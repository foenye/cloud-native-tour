package rest

import (
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration"
	apiregistrationv1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	aggregateorscheme "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apiserver/scheme"
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/registry/apiservice/etcd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericAPIServer "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
)

func NewRESTStorage(apiResourceConfigSource serverstorage.APIResourceConfigSource,
	restOptionsGetter generic.RESTOptionsGetter, _ bool, // shouldServeBata
) genericAPIServer.APIGroupInfo {
	apiGroupInfo := genericAPIServer.NewDefaultAPIGroupInfo(apiregistration.GroupName, aggregateorscheme.Scheme,
		metav1.ParameterCodec, aggregateorscheme.Codecs)
	storage := map[string]rest.Storage{}

	if resource := "apiservices"; apiResourceConfigSource.ResourceEnabled(apiregistrationv1.SchemeGroupVersion.
		WithResource(resource)) {
		apiServiceREST := etcd.NewREST(aggregateorscheme.Scheme, restOptionsGetter)
		storage[resource] = apiServiceREST
		storage[resource+"/status"] = etcd.NewStatusREST(aggregateorscheme.Scheme, apiServiceREST)
	}

	if len(storage) > 0 {
		apiGroupInfo.VersionedResourcesStorageMap["v1"] = storage
	}

	return apiGroupInfo
}
