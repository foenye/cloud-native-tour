package remote

import (
	"fmt"
	apiregistrationv1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/clientset_generated/clientset/fake"
	generatedClientsetTypedV1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
	generatedClientListersV1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/listers/apiregistration/v1"
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/controllers/status/metrics"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	kubeAPICoreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/dump"
	clientGoListersCoreV1 "k8s.io/client-go/listers/core/v1"
	clientGoTesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
)

const (
	testServicePort     int32 = 1234
	testServicePortName       = "testPort"
)

func newEndpoints(namespace, name string) *kubeAPICoreV1.Endpoints {
	return &kubeAPICoreV1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
}

func newEndpointsWithAddress(namespace, name string, port int32, portName string) *kubeAPICoreV1.Endpoints {
	return &kubeAPICoreV1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Subsets: []kubeAPICoreV1.EndpointSubset{
			{
				Addresses: []kubeAPICoreV1.EndpointAddress{
					{
						IP: "val",
					},
				},
				Ports: []kubeAPICoreV1.EndpointPort{
					{
						Name: portName,
						Port: port,
					},
				},
			},
		},
	}
}

func newService(namespace, name string, port int32, portName string) *kubeAPICoreV1.Service {
	return &kubeAPICoreV1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: kubeAPICoreV1.ServiceSpec{
			Type: kubeAPICoreV1.ServiceTypeClusterIP,
			Ports: []kubeAPICoreV1.ServicePort{
				{Port: port, Name: portName},
			},
		},
	}
}

func newLocalAPIService(name string) *apiregistrationv1.APIService {
	return &apiregistrationv1.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

func newRemoteAPIService(name string) *apiregistrationv1.APIService {
	return &apiregistrationv1.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: apiregistrationv1.APIServiceSpec{
			Group:   strings.SplitN(name, ".", 2)[0],
			Version: strings.SplitN(name, ".", 2)[1],
			Service: &apiregistrationv1.ServiceReference{
				Namespace: "foo",
				Name:      "bar",
				Port:      ptr.To(testServicePort),
			},
		},
	}
}

type T interface {
	Fatalf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

func setupAPIServices(t T, apiServices []runtime.Object) (*AvailableConditionController, *fake.Clientset) {
	//goland:noinspection GoDeprecation
	fakeClient := fake.NewSimpleClientset()
	apiServiceIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	serviceIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	endpointsIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	for _, o := range apiServices {
		if err := apiServiceIndexer.Add(o); err != nil {
			t.Fatalf("failed to add APIService: %v", err)
		}
	}

	c := AvailableConditionController{
		apiServiceClient: fakeClient.ApiregistrationV1(),
		apiServiceLister: generatedClientListersV1.NewAPIServiceLister(apiServiceIndexer),
		serviceLister:    clientGoListersCoreV1.NewServiceLister(serviceIndexer),
		endpointsLister:  clientGoListersCoreV1.NewEndpointsLister(endpointsIndexer),
		serviceResolver:  &fakeServiceResolver{url: testServer.URL},
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			// We want a fairly tight requeue time.  The controller listens to the API, but because it relies on the routability of the
			// service network, it is possible for an external, non-watchable factor to affect availability.  This keeps
			// the maximum disruption time to a minimum, but it does prevent hot loops.
			workqueue.NewTypedItemExponentialFailureRateLimiter[string](5*time.Millisecond, 30*time.Second),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "AvailableConditionController"},
		),
		metrics: metrics.New(),
	}
	for _, svc := range apiServices {
		c.addAPIService(svc)
	}
	return &c, fakeClient
}

func BenchmarkBuildCache(b *testing.B) {
	apiServiceName := "remote.group"
	// model 1 APIService pointing at a given service, and 30 pointing at local group/versions
	apiServices := []runtime.Object{newRemoteAPIService(apiServiceName)}
	for i := 0; i < 30; i++ {
		apiServices = append(apiServices, newLocalAPIService(fmt.Sprintf("local.group%d", i)))
	}
	// model one service backing an API service, and 100 unrelated services
	services := []*kubeAPICoreV1.Service{newService("foo", "bar", testServicePort, testServicePortName)}
	for i := 0; i < 100; i++ {
		services = append(services, newService("foo", fmt.Sprintf("bar%d", i), testServicePort, testServicePortName))
	}
	c, _ := setupAPIServices(b, apiServices)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 1; n <= b.N; n++ {
		for _, svc := range services {
			c.addService(svc)
		}
		for _, svc := range services {
			c.updateService(svc, svc)
		}
		for _, svc := range services {
			c.deleteService(svc)
		}
	}
}

func TestBuildCache(t *testing.T) {
	tests := []struct {
		name string

		apiServiceName string
		apiServices    []runtime.Object
		services       []*kubeAPICoreV1.Service
		endpoints      []*kubeAPICoreV1.Endpoints

		expectedAvailability apiregistrationv1.APIServiceCondition
	}{
		{
			name:           "api service",
			apiServiceName: "remote.group",
			apiServices:    []runtime.Object{newRemoteAPIService("remote.group")},
			services:       []*kubeAPICoreV1.Service{newService("foo", "bar", testServicePort, testServicePortName)},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, fakeClient := setupAPIServices(t, tc.apiServices)
			for _, svc := range tc.services {
				c.addService(svc)
			}

			_ = c.sync(tc.apiServiceName)

			// ought to have one action writing status
			if e, a := 1, len(fakeClient.Actions()); e != a {
				t.Fatalf("%v expected %v, got %v", tc.name, e, fakeClient.Actions())
			}
		})
	}
}

func TestSync(t *testing.T) {
	tests := []struct {
		name string

		apiServiceName  string
		apiServices     []runtime.Object
		services        []*kubeAPICoreV1.Service
		endpoints       []*kubeAPICoreV1.Endpoints
		backendStatus   int
		backendLocation string

		expectedAvailability apiregistrationv1.APIServiceCondition
		expectedSyncError    string
		expectedSkipped      bool
	}{
		{
			name:           "local",
			apiServiceName: "local.group",
			apiServices:    []runtime.Object{newLocalAPIService("local.group")},
			backendStatus:  http.StatusOK,
			expectedAvailability: apiregistrationv1.APIServiceCondition{
				Type:    apiregistrationv1.Available,
				Status:  apiregistrationv1.ConditionTrue,
				Reason:  "Local",
				Message: "Local APIServices are always available",
			},
			expectedSkipped: true,
		},
		{
			name:           "no service",
			apiServiceName: "remote.group",
			apiServices:    []runtime.Object{newRemoteAPIService("remote.group")},
			services:       []*kubeAPICoreV1.Service{newService("foo", "not-bar", testServicePort, testServicePortName)},
			backendStatus:  http.StatusOK,
			expectedAvailability: apiregistrationv1.APIServiceCondition{
				Type:    apiregistrationv1.Available,
				Status:  apiregistrationv1.ConditionFalse,
				Reason:  "ServiceNotFound",
				Message: `service/bar in "foo" is not present`,
			},
		},
		{
			name:           "service on bad port",
			apiServiceName: "remote.group",
			apiServices:    []runtime.Object{newRemoteAPIService("remote.group")},
			services: []*kubeAPICoreV1.Service{{
				ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"},
				Spec: kubeAPICoreV1.ServiceSpec{
					Type: kubeAPICoreV1.ServiceTypeClusterIP,
					Ports: []kubeAPICoreV1.ServicePort{
						{Port: 6443},
					},
				},
			}},
			endpoints:     []*kubeAPICoreV1.Endpoints{newEndpointsWithAddress("foo", "bar", testServicePort, testServicePortName)},
			backendStatus: http.StatusOK,
			expectedAvailability: apiregistrationv1.APIServiceCondition{
				Type:    apiregistrationv1.Available,
				Status:  apiregistrationv1.ConditionFalse,
				Reason:  "ServicePortError",
				Message: fmt.Sprintf(`service/bar in "foo" is not listening on port %d`, testServicePort),
			},
		},
		{
			name:           "no endpoints",
			apiServiceName: "remote.group",
			apiServices:    []runtime.Object{newRemoteAPIService("remote.group")},
			services:       []*kubeAPICoreV1.Service{newService("foo", "bar", testServicePort, testServicePortName)},
			backendStatus:  http.StatusOK,
			expectedAvailability: apiregistrationv1.APIServiceCondition{
				Type:    apiregistrationv1.Available,
				Status:  apiregistrationv1.ConditionFalse,
				Reason:  "EndpointsNotFound",
				Message: `cannot find endpoints for service/bar in "foo"`,
			},
		},
		{
			name:           "missing endpoints",
			apiServiceName: "remote.group",
			apiServices:    []runtime.Object{newRemoteAPIService("remote.group")},
			services:       []*kubeAPICoreV1.Service{newService("foo", "bar", testServicePort, testServicePortName)},
			endpoints:      []*kubeAPICoreV1.Endpoints{newEndpoints("foo", "bar")},
			backendStatus:  http.StatusOK,
			expectedAvailability: apiregistrationv1.APIServiceCondition{
				Type:    apiregistrationv1.Available,
				Status:  apiregistrationv1.ConditionFalse,
				Reason:  "MissingEndpoints",
				Message: `endpoints for service/bar in "foo" have no addresses with port name "testPort"`,
			},
		},
		{
			name:           "wrong endpoint port name",
			apiServiceName: "remote.group",
			apiServices:    []runtime.Object{newRemoteAPIService("remote.group")},
			services:       []*kubeAPICoreV1.Service{newService("foo", "bar", testServicePort, testServicePortName)},
			endpoints:      []*kubeAPICoreV1.Endpoints{newEndpointsWithAddress("foo", "bar", testServicePort, "wrongName")},
			backendStatus:  http.StatusOK,
			expectedAvailability: apiregistrationv1.APIServiceCondition{
				Type:    apiregistrationv1.Available,
				Status:  apiregistrationv1.ConditionFalse,
				Reason:  "MissingEndpoints",
				Message: fmt.Sprintf(`endpoints for service/bar in "foo" have no addresses with port name "%s"`, testServicePortName),
			},
		},
		{
			name:           "remote",
			apiServiceName: "remote.group",
			apiServices:    []runtime.Object{newRemoteAPIService("remote.group")},
			services:       []*kubeAPICoreV1.Service{newService("foo", "bar", testServicePort, testServicePortName)},
			endpoints:      []*kubeAPICoreV1.Endpoints{newEndpointsWithAddress("foo", "bar", testServicePort, testServicePortName)},
			backendStatus:  http.StatusOK,
			expectedAvailability: apiregistrationv1.APIServiceCondition{
				Type:    apiregistrationv1.Available,
				Status:  apiregistrationv1.ConditionTrue,
				Reason:  "Passed",
				Message: `all checks passed`,
			},
		},
		{
			name:           "remote-bad-return",
			apiServiceName: "remote.group",
			apiServices:    []runtime.Object{newRemoteAPIService("remote.group")},
			services:       []*kubeAPICoreV1.Service{newService("foo", "bar", testServicePort, testServicePortName)},
			endpoints:      []*kubeAPICoreV1.Endpoints{newEndpointsWithAddress("foo", "bar", testServicePort, testServicePortName)},
			backendStatus:  http.StatusForbidden,
			expectedAvailability: apiregistrationv1.APIServiceCondition{
				Type:    apiregistrationv1.Available,
				Status:  apiregistrationv1.ConditionFalse,
				Reason:  "FailedDiscoveryCheck",
				Message: `failing or missing response from`,
			},
			expectedSyncError: "failing or missing response from",
		},
		{
			name:            "remote-redirect",
			apiServiceName:  "remote.group",
			apiServices:     []runtime.Object{newRemoteAPIService("remote.group")},
			services:        []*kubeAPICoreV1.Service{newService("foo", "bar", testServicePort, testServicePortName)},
			endpoints:       []*kubeAPICoreV1.Endpoints{newEndpointsWithAddress("foo", "bar", testServicePort, testServicePortName)},
			backendStatus:   http.StatusFound,
			backendLocation: "/test",
			expectedAvailability: apiregistrationv1.APIServiceCondition{
				Type:    apiregistrationv1.Available,
				Status:  apiregistrationv1.ConditionFalse,
				Reason:  "FailedDiscoveryCheck",
				Message: `failing or missing response from`,
			},
			expectedSyncError: "failing or missing response from",
		},
		{
			name:           "remote-304",
			apiServiceName: "remote.group",
			apiServices:    []runtime.Object{newRemoteAPIService("remote.group")},
			services:       []*kubeAPICoreV1.Service{newService("foo", "bar", testServicePort, testServicePortName)},
			endpoints:      []*kubeAPICoreV1.Endpoints{newEndpointsWithAddress("foo", "bar", testServicePort, testServicePortName)},
			backendStatus:  http.StatusNotModified,
			expectedAvailability: apiregistrationv1.APIServiceCondition{
				Type:    apiregistrationv1.Available,
				Status:  apiregistrationv1.ConditionFalse,
				Reason:  "FailedDiscoveryCheck",
				Message: `failing or missing response from`,
			},
			expectedSyncError: "failing or missing response from",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			//goland:noinspection GoDeprecation
			fakeClient := fake.NewSimpleClientset(tc.apiServices...)
			apiServiceIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			serviceIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			endpointsIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			for _, obj := range tc.apiServices {
				_ = apiServiceIndexer.Add(obj)
			}
			for _, obj := range tc.services {
				_ = serviceIndexer.Add(obj)
			}
			for _, obj := range tc.endpoints {
				_ = endpointsIndexer.Add(obj)
			}

			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.backendLocation != "" {
					w.Header().Set("Location", tc.backendLocation)
				}
				w.WriteHeader(tc.backendStatus)
			}))
			defer testServer.Close()

			c := AvailableConditionController{
				apiServiceClient:           fakeClient.ApiregistrationV1(),
				apiServiceLister:           generatedClientListersV1.NewAPIServiceLister(apiServiceIndexer),
				serviceLister:              clientGoListersCoreV1.NewServiceLister(serviceIndexer),
				endpointsLister:            clientGoListersCoreV1.NewEndpointsLister(endpointsIndexer),
				serviceResolver:            &fakeServiceResolver{url: testServer.URL},
				proxyCurrentCertKeyContent: func() ([]byte, []byte) { return emptyCert(), emptyCert() },
				metrics:                    metrics.New(),
			}
			err := c.sync(tc.apiServiceName)
			if tc.expectedSyncError != "" {
				if err == nil {
					t.Fatalf("%v expected error with %q, got none", tc.name, tc.expectedSyncError)
				} else if !strings.Contains(err.Error(), tc.expectedSyncError) {
					t.Fatalf("%v expected error with %q, got %q", tc.name, tc.expectedSyncError, err.Error())
				}
			} else if err != nil {
				t.Fatalf("%v unexpected sync error: %v", tc.name, err)
			}

			if tc.expectedSkipped {
				if len(fakeClient.Actions()) > 0 {
					t.Fatalf("%v expected no actions, got %v", tc.name, fakeClient.Actions())
				}
				return
			}

			// ought to have one action writing status
			if e, a := 1, len(fakeClient.Actions()); e != a {
				t.Fatalf("%v expected %v, got %v", tc.name, e, fakeClient.Actions())
			}

			action, ok := fakeClient.Actions()[0].(clientGoTesting.UpdateAction)
			if !ok {
				t.Fatalf("%v got %v", tc.name, ok)
			}

			if e, a := 1, len(action.GetObject().(*apiregistrationv1.APIService).Status.Conditions); e != a {
				t.Fatalf("%v expected %v, got %v", tc.name, e, action.GetObject())
			}
			condition := action.GetObject().(*apiregistrationv1.APIService).Status.Conditions[0]
			if e, a := tc.expectedAvailability.Type, condition.Type; e != a {
				t.Errorf("%v expected %v, got %#v", tc.name, e, condition)
			}
			if e, a := tc.expectedAvailability.Status, condition.Status; e != a {
				t.Errorf("%v expected %v, got %#v", tc.name, e, condition)
			}
			if e, a := tc.expectedAvailability.Reason, condition.Reason; e != a {
				t.Errorf("%v expected %v, got %#v", tc.name, e, condition)
			}
			if e, a := tc.expectedAvailability.Message, condition.Message; !strings.HasPrefix(a, e) {
				t.Errorf("%v expected %v, got %#v", tc.name, e, condition)
			}
			if condition.LastTransitionTime.IsZero() {
				t.Error("expected lastTransitionTime to be non-zero")
			}
		})
	}
}

type fakeServiceResolver struct {
	url string
}

func (f *fakeServiceResolver) ResolveEndpoint(_, _ string, _ int32) (*url.URL, error) {
	return url.Parse(f.url)
}

func TestUpdateAPIServiceStatus(t *testing.T) {
	foo := &apiregistrationv1.APIService{Status: apiregistrationv1.APIServiceStatus{Conditions: []apiregistrationv1.APIServiceCondition{{Type: "foo"}}}}
	bar := &apiregistrationv1.APIService{Status: apiregistrationv1.APIServiceStatus{Conditions: []apiregistrationv1.APIServiceCondition{{Type: "bar"}}}}

	//goland:noinspection GoDeprecation
	fakeClient := fake.NewSimpleClientset(foo)
	c := AvailableConditionController{
		apiServiceClient: fakeClient.ApiregistrationV1().(generatedClientsetTypedV1.APIServicesGetter),
		metrics:          metrics.New(),
	}

	if _, err := c.updateAPIServiceStatus(foo, foo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e, a := 0, len(fakeClient.Actions()); e != a {
		t.Error(dump.Pretty(fakeClient.Actions()))
	}

	fakeClient.ClearActions()
	if _, err := c.updateAPIServiceStatus(foo, bar); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e, a := 1, len(fakeClient.Actions()); e != a {
		t.Error(dump.Pretty(fakeClient.Actions()))
	}
}

func emptyCert() []byte {
	return []byte{}
}
