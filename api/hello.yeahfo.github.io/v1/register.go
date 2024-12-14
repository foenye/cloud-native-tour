package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is group version used to register these Objects.
var SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

var (
	SchemeBuilder      runtime.SchemeBuilder
	localSchemeBuilder = &SchemeBuilder
	// AddToScheme adds this group to a scheme.
	AddToScheme = localSchemeBuilder.AddToScheme
)

func init() {
	localSchemeBuilder.Register(addKnownTypes, RegisterDefaults)
}

// Adds the list of known types to the given scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion, &Foo{}, &FooList{})
	return nil
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(Resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(Resource).GroupResource()
}
