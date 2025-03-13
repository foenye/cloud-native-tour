package hello_eonvon_github_io

import (
	v1 "github.com/eonvon/cloud-native-tour/api/hello.eonvon.github.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is the hub group version for all kinds.
var SchemeGroupVersion = schema.GroupVersion{Group: v1.Group, Version: runtime.APIVersionInternal}

var (
	SchemeBuilder      runtime.SchemeBuilder
	localSchemeBuilder = &SchemeBuilder

	// AddToScheme adds this group to a scheme.
	AddToScheme = localSchemeBuilder.AddToScheme
)

func init() {
	localSchemeBuilder.Register(addKnownTypes)
}

// Adds the list of known hub types to the given scheme.
// use v1 as hub type for now
//
// Normally tree of api types should like
// - hello.eonvon.github.io
//   - v1beta1/{types.go, register.go}
//   - v1/{types.go, register.go}
//   - types.go, register.go             <--- hub types and register
//
// all types of different kind should be translate to the internal hub type in memory
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&v1.Foo{},
		&v1.FooList{},
	)
	metav1.AddToGroupVersion(scheme, v1.SchemeGroupVersion)
	return nil
}
