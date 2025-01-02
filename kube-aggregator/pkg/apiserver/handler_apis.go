package apiserver

import (
	apiregistrationv1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1helper "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1/helper"
	apiregistrationv1beta1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	generatedClientListersV1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/listers/apiregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"net/http"
)

var _ http.Handler = &apisHandler{}

// apiHandler serves the `/apis` endpoint.
// This registered as a filter so that it never collides with any explicitly registered endpoints
type apisHandler struct {
	codecs         serializer.CodecFactory
	lister         generatedClientListersV1.APIServiceLister
	discoveryGroup metav1.APIGroup
}

func (handler *apisHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	discoveryGroupList := &metav1.APIGroupList{
		// Always add our API group to list first.  Since we'll never hava a registered APIService for it and since
		// this is the crux of the API, having this first will give our names priority. It's good be king.
		Groups: []metav1.APIGroup{handler.discoveryGroup},
	}

	apiServices, err := handler.lister.List(labels.Everything())
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	apiServicesByGroup := apiregistrationv1helper.SortedByGroupAndVersion(apiServices)
	for _, apiGroupServers := range apiServicesByGroup {
		// skip the legacy group
		if len(apiGroupServers[0].Spec.Group) == 0 {
			continue
		}
		if discoveryGroup := convertToDiscoveryAPIGroup(apiGroupServers); discoveryGroup != nil {
			discoveryGroupList.Groups = append(discoveryGroupList.Groups, *discoveryGroup)
		}
	}
	responsewriters.WriteObjectNegotiated(handler.codecs, negotiation.DefaultEndpointRestrictions, schema.GroupVersion{},
		response, request, http.StatusOK, discoveryGroupList, false)
}

func discoveryGroup(enabledVersions sets.Set[string]) metav1.APIGroup {
	discoveryGroup := metav1.APIGroup{
		Name: apiregistrationv1.GroupName,
		Versions: []metav1.GroupVersionForDiscovery{
			{
				GroupVersion: apiregistrationv1.SchemeGroupVersion.String(),
				Version:      apiregistrationv1.SchemeGroupVersion.Version,
			},
		},
		PreferredVersion: metav1.GroupVersionForDiscovery{
			GroupVersion: apiregistrationv1.SchemeGroupVersion.String(),
			Version:      apiregistrationv1.SchemeGroupVersion.Version,
		},
	}

	if enabledVersions.Has(apiregistrationv1beta1.SchemeGroupVersion.Version) {
		discoveryGroup.Versions = append(discoveryGroup.Versions, metav1.GroupVersionForDiscovery{
			GroupVersion: apiregistrationv1beta1.SchemeGroupVersion.String(),
			Version:      apiregistrationv1beta1.SchemeGroupVersion.Version,
		})
	}

	return discoveryGroup
}

// convertToDiscoveryAPIGroup takes api services in a single group and returns a discovery compatible object.
// if none of the services are available, it will return nil.
func convertToDiscoveryAPIGroup(apiServices []*apiregistrationv1.APIService) *metav1.APIGroup {
	apiServicesByGroup := apiregistrationv1helper.SortedByGroupAndVersion(apiServices)[0]

	var discoveryGroup *metav1.APIGroup

	for _, apiService := range apiServicesByGroup {
		// the first APIService which is valid becomes the default
		if discoveryGroup == nil {
			discoveryGroup = &metav1.APIGroup{
				Name: apiService.Spec.Group,
				PreferredVersion: metav1.GroupVersionForDiscovery{
					GroupVersion: apiService.Spec.Group + "/" + apiService.Spec.Version,
					Version:      apiService.Spec.Version,
				},
			}
		}

		discoveryGroup.Versions = append(discoveryGroup.Versions,
			metav1.GroupVersionForDiscovery{
				GroupVersion: apiService.Spec.Group + "/" + apiService.Spec.Version,
				Version:      apiService.Spec.Version,
			},
		)
	}

	return discoveryGroup
}

var _ http.Handler = &apiGroupHandler{}

// apiGroupHandler serves the `/apis/<group>` endpoint.
type apiGroupHandler struct {
	codecs    serializer.CodecFactory
	groupName string

	lister generatedClientListersV1.APIServiceLister

	delegate http.Handler
}

func (handler *apiGroupHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	apiServices, err := handler.lister.List(labels.Everything())
	if //goland:noinspection GoTypeAssertionOnErrors
	statusErr, ok := err.(*errors.StatusError); ok {
		responsewriters.WriteRawJSON(int(statusErr.Status().Code), statusErr.Status(), response)
		return
	}
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	var apiServicesForGroup []*apiregistrationv1.APIService
	for _, apiService := range apiServices {
		if apiService.Spec.Group == handler.groupName {
			apiServicesForGroup = append(apiServicesForGroup, apiService)
		}
	}

	if len(apiServicesForGroup) == 0 {
		handler.delegate.ServeHTTP(response, request)
		return
	}

	discoveryGroup := convertToDiscoveryAPIGroup(apiServicesForGroup)
	if discoveryGroup == nil {
		http.Error(response, "", http.StatusNotFound)
		return
	}
	responsewriters.WriteObjectNegotiated(handler.codecs, negotiation.DefaultEndpointRestrictions, schema.GroupVersion{},
		response, request, http.StatusOK, discoveryGroup, false)
}
