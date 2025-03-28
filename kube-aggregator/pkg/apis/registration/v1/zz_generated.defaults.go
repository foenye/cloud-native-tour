//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by defaulter-gen. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&APIService{}, func(obj interface{}) { SetObjectDefaults_APIService(obj.(*APIService)) })
	scheme.AddTypeDefaultingFunc(&APIServiceList{}, func(obj interface{}) { SetObjectDefaults_APIServiceList(obj.(*APIServiceList)) })
	return nil
}

func SetObjectDefaults_APIService(in *APIService) {
	if in.Spec.Service != nil {
		SetDefaults_ServiceReference(in.Spec.Service)
	}
}

func SetObjectDefaults_APIServiceList(in *APIServiceList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_APIService(a)
	}
}
