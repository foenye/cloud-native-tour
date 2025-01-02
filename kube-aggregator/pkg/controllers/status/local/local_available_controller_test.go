package local

import (
	apiregistrationv1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration/v1"
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/clientset_generated/clientset/fake"
	generatedClientsetTypedV1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
	generatedClientListersV1 "github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/client/listers/apiregistration/v1"
	"github.com/yeahfo/cloud-native-tour/kube-aggregator/pkg/controllers/status/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/dump"
	clientGoTesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/ptr"
	"strings"
	"testing"
)

const (
	testServicePort int32 = 1234
)

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

func TestSync(t *testing.T) {
	tests := []struct {
		name string

		apiServiceName string
		apiServices    []runtime.Object

		expectedAvailability apiregistrationv1.APIServiceCondition
		expectedAction       bool
	}{
		{
			name:           "local",
			apiServiceName: "local.group",
			apiServices:    []runtime.Object{newLocalAPIService("local.group")},
			expectedAvailability: apiregistrationv1.APIServiceCondition{
				Type:    apiregistrationv1.Available,
				Status:  apiregistrationv1.ConditionTrue,
				Reason:  "Local",
				Message: "Local APIServices are always available",
			},
			expectedAction: true,
		},
		{
			name:           "remote",
			apiServiceName: "remote.group",
			apiServices:    []runtime.Object{newRemoteAPIService("remote.group")},
			expectedAction: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			//goland:noinspection GoDeprecation
			fakeClient := fake.NewSimpleClientset(tc.apiServices...)
			apiServiceIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			for _, obj := range tc.apiServices {
				if err := apiServiceIndexer.Add(obj); err != nil {
					t.Fatalf("failed to add object to indexer: %v", err)
				}
			}

			c := AvailableConditionController{
				apiServiceClient: fakeClient.ApiregistrationV1(),
				apiServiceLister: generatedClientListersV1.NewAPIServiceLister(apiServiceIndexer),
				metrics:          metrics.New(),
			}
			if err := c.sync(tc.apiServiceName); err != nil {
				t.Fatalf("unexpect sync error: %v", err)
			}

			// ought to have one action writing status
			if e, a := tc.expectedAction, len(fakeClient.Actions()) == 1; e != a {
				t.Fatalf("%v expected %v, got %v", tc.name, e, fakeClient.Actions())
			}
			if tc.expectedAction {
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
			}
		})
	}
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
		t.Fatalf("unexpected updateAPIServiceStatus error: %v", err)
	}
	if e, a := 0, len(fakeClient.Actions()); e != a {
		t.Error(dump.Pretty(fakeClient.Actions()))
	}

	fakeClient.ClearActions()
	if _, err := c.updateAPIServiceStatus(foo, bar); err != nil {
		t.Fatalf("unexpected updateAPIServiceStatus error: %v", err)
	}
	if e, a := 1, len(fakeClient.Actions()); e != a {
		t.Error(dump.Pretty(fakeClient.Actions()))
	}
}
