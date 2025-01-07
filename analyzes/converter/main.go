package main

import (
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/apis/apps"
)

func main() {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(appsv1beta1.SchemeGroupVersion, &appsv1beta1.Deployment{})
	scheme.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
	scheme.AddKnownTypes(apps.SchemeGroupVersion, &appsv1.Deployment{})

	metav1.AddToGroupVersion(scheme, appsv1.SchemeGroupVersion)
	metav1.AddToGroupVersion(scheme, appsv1beta1.SchemeGroupVersion)
}
