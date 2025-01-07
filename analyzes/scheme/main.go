package main

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func main() {
	// KnownType external
	coreGV := schema.GroupVersion{Group: "", Version: "v1"}
	extensionsGV := schema.GroupVersion{Group: "extensions", Version: "v1beta1"}

	// KnownType internal
	coreInternalGV := schema.GroupVersion{Group: "", Version: runtime.APIVersionInternal}

	// UnversionedType
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(coreGV, &corev1.Pod{})
	scheme.AddKnownTypes(extensionsGV, &appsv1.Deployment{}, &appsv1.DaemonSet{})
	scheme.AddKnownTypes(coreInternalGV, &corev1.Service{})
	scheme.AddUnversionedTypes(unversioned, &metav1.Status{}, &metav1.APIGroup{})
	println(scheme.IsGroupRegistered("extensions"), scheme.IsVersionRegistered(coreGV))
	for kind, ref := range scheme.AllKnownTypes() {
		println(kind.String(), ref.String())
	}
}
