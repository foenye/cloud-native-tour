// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "github.com/foenye/cloud-native-tour/crd-getting-started/pkg/client/clientset_generated/clientset/typed/greeting/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeGreetingV1 struct {
	*testing.Fake
}

func (c *FakeGreetingV1) Foos(namespace string) v1.FooInterface {
	return newFakeFoos(c, namespace)
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeGreetingV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
