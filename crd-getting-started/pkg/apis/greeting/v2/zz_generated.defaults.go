//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by defaulter-gen. DO NOT EDIT.

package v2

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&Foo{}, func(obj interface{}) { SetObjectDefaults_Foo(obj.(*Foo)) })
	scheme.AddTypeDefaultingFunc(&FooList{}, func(obj interface{}) { SetObjectDefaults_FooList(obj.(*FooList)) })
	return nil
}

func SetObjectDefaults_Foo(in *Foo) {
	SetDefaults_Foo(in)
}

func SetObjectDefaults_FooList(in *FooList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Foo(a)
	}
}
