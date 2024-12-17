package v1beta1

import (
	transformationv1beta1 "github.com/yeahfo/cloud-native-tour/api/transformation/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName specifies the group name used to register the objects.
const GroupName = "transformation.yeahfo.github.io"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1beta1"}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	localSchemeBuilder = &transformationv1beta1.SchemeBuilder
	AddToScheme        = localSchemeBuilder.AddToScheme
)
