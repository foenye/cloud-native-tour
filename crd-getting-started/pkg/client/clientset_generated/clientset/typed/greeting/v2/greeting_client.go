// Code generated by client-gen. DO NOT EDIT.

package v2

import (
	http "net/http"

	greetingv2 "github.com/foenye/cloud-native-tour/crd-getting-started/pkg/apis/greeting/v2"
	scheme "github.com/foenye/cloud-native-tour/crd-getting-started/pkg/client/clientset_generated/clientset/scheme"
	rest "k8s.io/client-go/rest"
)

type GreetingV2Interface interface {
	RESTClient() rest.Interface
	FoosGetter
}

// GreetingV2Client is used to interact with features provided by the greeting.foen.ye group.
type GreetingV2Client struct {
	restClient rest.Interface
}

func (c *GreetingV2Client) Foos(namespace string) FooInterface {
	return newFoos(c, namespace)
}

// NewForConfig creates a new GreetingV2Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*GreetingV2Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new GreetingV2Client for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*GreetingV2Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &GreetingV2Client{client}, nil
}

// NewForConfigOrDie creates a new GreetingV2Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *GreetingV2Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new GreetingV2Client for the given RESTClient.
func New(c rest.Interface) *GreetingV2Client {
	return &GreetingV2Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := greetingv2.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = rest.CodecFactoryForGeneratedClient(scheme.Scheme, scheme.Codecs).WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *GreetingV2Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
