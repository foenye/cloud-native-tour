// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	greetingv1 "github.com/foenye/cloud-native-tour/crd-getting-started/pkg/apis/greeting/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// FooLister helps list Foos.
// All objects returned here must be treated as read-only.
type FooLister interface {
	// List lists all Foos in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*greetingv1.Foo, err error)
	// Foos returns an object that can list and get Foos.
	Foos(namespace string) FooNamespaceLister
	FooListerExpansion
}

// fooLister implements the FooLister interface.
type fooLister struct {
	listers.ResourceIndexer[*greetingv1.Foo]
}

// NewFooLister returns a new FooLister.
func NewFooLister(indexer cache.Indexer) FooLister {
	return &fooLister{listers.New[*greetingv1.Foo](indexer, greetingv1.Resource("foo"))}
}

// Foos returns an object that can list and get Foos.
func (s *fooLister) Foos(namespace string) FooNamespaceLister {
	return fooNamespaceLister{listers.NewNamespaced[*greetingv1.Foo](s.ResourceIndexer, namespace)}
}

// FooNamespaceLister helps list and get Foos.
// All objects returned here must be treated as read-only.
type FooNamespaceLister interface {
	// List lists all Foos in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*greetingv1.Foo, err error)
	// Get retrieves the Foo from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*greetingv1.Foo, error)
	FooNamespaceListerExpansion
}

// fooNamespaceLister implements the FooNamespaceLister
// interface.
type fooNamespaceLister struct {
	listers.ResourceIndexer[*greetingv1.Foo]
}
