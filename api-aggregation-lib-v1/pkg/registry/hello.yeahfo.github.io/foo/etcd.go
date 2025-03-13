package foo

import (
	hellov1 "github.com/eonvon/cloud-native-tour/api/hello.eonvon.github.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/generic/registry"
)

func NewREST(scheme *runtime.Scheme, optionsGetter generic.RESTOptionsGetter) (*registry.Store, error) {
	strategy := newStrategy(scheme)

	store := &registry.Store{
		NewFunc: func() runtime.Object {
			return &hellov1.Foo{}
		},
		NewListFunc: func() runtime.Object {
			return &hellov1.FooList{}
		},
		PredicateFunc:             matchFoo,
		DefaultQualifiedResource:  hellov1.Resource(hellov1.Plural),
		SingularQualifiedResource: hellov1.Resource(hellov1.Plural),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,
		TableConvertor: strategy,
	}

	options := &generic.StoreOptions{RESTOptions: optionsGetter, AttrFunc: getAttributes}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}
	return store, nil
}
