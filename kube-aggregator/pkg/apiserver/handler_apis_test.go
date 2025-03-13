package apiserver

import (
	"github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apis/apiregistration"
	apiserverSchema "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/apiserver/scheme"
	generatedClientListersV1 "github.com/eonvon/cloud-native-tour/kube-aggregator/pkg/client/listers/apiregistration/v1"
	"github.com/google/go-cmp/cmp"
	"io"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"
)

func TestAPIs(t *testing.T) {
	tests := []struct {
		name        string
		enabled     sets.Set[string]
		apiservices []*apiregistration.APIService
		expected    *metav1.APIGroupList
	}{
		{
			name:        "empty",
			enabled:     sets.New("v1", "v1beta1"),
			apiservices: []*apiregistration.APIService{},
			expected: &metav1.APIGroupList{
				TypeMeta: metav1.TypeMeta{Kind: "APIGroupList", APIVersion: "v1"},
				Groups: []metav1.APIGroup{
					discoveryGroup(sets.New("v1", "v1beta1")),
				},
			},
		},
		{
			name:        "v1 only",
			enabled:     sets.New("v1"),
			apiservices: []*apiregistration.APIService{},
			expected: &metav1.APIGroupList{
				TypeMeta: metav1.TypeMeta{Kind: "APIGroupList", APIVersion: "v1"},
				Groups: []metav1.APIGroup{
					discoveryGroup(sets.New("v1")),
				},
			},
		},
		{
			name:    "simple add",
			enabled: sets.New("v1", "v1beta1"),
			apiservices: []*apiregistration.APIService{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v1.foo"},
					Spec: apiregistration.APIServiceSpec{
						Service: &apiregistration.ServiceReference{
							Namespace: "ns",
							Name:      "api",
						},
						Group:                "foo",
						Version:              "v1",
						GroupPriorityMinimum: 11,
					},
					Status: apiregistration.APIServiceStatus{
						Conditions: []apiregistration.APIServiceCondition{
							{Type: apiregistration.Available, Status: apiregistration.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v1.bar"},
					Spec: apiregistration.APIServiceSpec{
						Service: &apiregistration.ServiceReference{
							Namespace: "ns",
							Name:      "api",
						},
						Group:                "bar",
						Version:              "v1",
						GroupPriorityMinimum: 10,
					},
					Status: apiregistration.APIServiceStatus{
						Conditions: []apiregistration.APIServiceCondition{
							{Type: apiregistration.Available, Status: apiregistration.ConditionTrue},
						},
					},
				},
			},
			expected: &metav1.APIGroupList{
				TypeMeta: metav1.TypeMeta{Kind: "APIGroupList", APIVersion: "v1"},
				Groups: []metav1.APIGroup{
					discoveryGroup(sets.New("v1", "v1beta1")),
					{
						Name: "foo",
						Versions: []metav1.GroupVersionForDiscovery{
							{
								GroupVersion: "foo/v1",
								Version:      "v1",
							},
						},
						PreferredVersion: metav1.GroupVersionForDiscovery{
							GroupVersion: "foo/v1",
							Version:      "v1",
						},
					},
					{
						Name: "bar",
						Versions: []metav1.GroupVersionForDiscovery{
							{
								GroupVersion: "bar/v1",
								Version:      "v1",
							},
						},
						PreferredVersion: metav1.GroupVersionForDiscovery{
							GroupVersion: "bar/v1",
							Version:      "v1",
						},
					},
				},
			},
		},
		{
			name:    "sorting",
			enabled: sets.New("v1", "v1beta1"),
			apiservices: []*apiregistration.APIService{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v1.foo"},
					Spec: apiregistration.APIServiceSpec{
						Service: &apiregistration.ServiceReference{
							Namespace: "ns",
							Name:      "api",
						},
						Group:                "foo",
						Version:              "v1",
						GroupPriorityMinimum: 20,
						VersionPriority:      10,
					},
					Status: apiregistration.APIServiceStatus{
						Conditions: []apiregistration.APIServiceCondition{
							{Type: apiregistration.Available, Status: apiregistration.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v2.bar"},
					Spec: apiregistration.APIServiceSpec{
						Service: &apiregistration.ServiceReference{
							Namespace: "ns",
							Name:      "api",
						},
						Group:                "bar",
						Version:              "v2",
						GroupPriorityMinimum: 11,
					},
					Status: apiregistration.APIServiceStatus{
						Conditions: []apiregistration.APIServiceCondition{
							{Type: apiregistration.Available, Status: apiregistration.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v2.foo"},
					Spec: apiregistration.APIServiceSpec{
						Service: &apiregistration.ServiceReference{
							Namespace: "ns",
							Name:      "api",
						},
						Group:                "foo",
						Version:              "v2",
						GroupPriorityMinimum: 1,
						VersionPriority:      15,
					},
					Status: apiregistration.APIServiceStatus{
						Conditions: []apiregistration.APIServiceCondition{
							{Type: apiregistration.Available, Status: apiregistration.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v1.bar"},
					Spec: apiregistration.APIServiceSpec{
						Service: &apiregistration.ServiceReference{
							Namespace: "ns",
							Name:      "api",
						},
						Group:                "bar",
						Version:              "v1",
						GroupPriorityMinimum: 11,
					},
					Status: apiregistration.APIServiceStatus{
						Conditions: []apiregistration.APIServiceCondition{
							{Type: apiregistration.Available, Status: apiregistration.ConditionTrue},
						},
					},
				},
			},
			expected: &metav1.APIGroupList{
				TypeMeta: metav1.TypeMeta{Kind: "APIGroupList", APIVersion: "v1"},
				Groups: []metav1.APIGroup{
					discoveryGroup(sets.New("v1", "v1beta1")),
					{
						Name: "foo",
						Versions: []metav1.GroupVersionForDiscovery{
							{
								GroupVersion: "foo/v2",
								Version:      "v2",
							},
							{
								GroupVersion: "foo/v1",
								Version:      "v1",
							},
						},
						PreferredVersion: metav1.GroupVersionForDiscovery{
							GroupVersion: "foo/v2",
							Version:      "v2",
						},
					},
					{
						Name: "bar",
						Versions: []metav1.GroupVersionForDiscovery{
							{
								GroupVersion: "bar/v2",
								Version:      "v2",
							},
							{
								GroupVersion: "bar/v1",
								Version:      "v1",
							},
						},
						PreferredVersion: metav1.GroupVersionForDiscovery{
							GroupVersion: "bar/v2",
							Version:      "v2",
						},
					},
				},
			},
		},
		{
			name:    "unavailable service",
			enabled: sets.New("v1", "v1beta1"),
			apiservices: []*apiregistration.APIService{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v1.foo"},
					Spec: apiregistration.APIServiceSpec{
						Service: &apiregistration.ServiceReference{
							Namespace: "ns",
							Name:      "api",
						},
						Group:                "foo",
						Version:              "v1",
						GroupPriorityMinimum: 11,
					},
					Status: apiregistration.APIServiceStatus{
						Conditions: []apiregistration.APIServiceCondition{
							{Type: apiregistration.Available, Status: apiregistration.ConditionFalse},
						},
					},
				},
			},
			expected: &metav1.APIGroupList{
				TypeMeta: metav1.TypeMeta{Kind: "APIGroupList", APIVersion: "v1"},
				Groups: []metav1.APIGroup{
					discoveryGroup(sets.New("v1", "v1beta1")),
					{
						Name: "foo",
						Versions: []metav1.GroupVersionForDiscovery{
							{
								GroupVersion: "foo/v1",
								Version:      "v1",
							},
						},
						PreferredVersion: metav1.GroupVersionForDiscovery{
							GroupVersion: "foo/v1",
							Version:      "v1",
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		handler := &apisHandler{
			codecs:         apiserverSchema.Codecs,
			lister:         generatedClientListersV1.NewAPIServiceLister(indexer),
			discoveryGroup: discoveryGroup(tc.enabled),
		}
		for _, o := range tc.apiservices {
			_ = indexer.Add(o)
		}

		server := httptest.NewServer(handler)
		//goland:noinspection GoDeferInLoop
		defer server.Close()

		resp, err := http.Get(server.URL + "/apis")
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}

		actual := &metav1.APIGroupList{}
		if err := runtime.DecodeInto(apiserverSchema.Codecs.UniversalDecoder(), bytes, actual); err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if !equality.Semantic.DeepEqual(tc.expected, actual) {
			t.Errorf("%s: %v", tc.name, cmp.Diff(tc.expected, actual))
			continue
		}
	}
}

func TestAPIGroupMissing(t *testing.T) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	handler := &apiGroupHandler{
		codecs:    apiserverSchema.Codecs,
		lister:    generatedClientListersV1.NewAPIServiceLister(indexer),
		groupName: "groupName",
		delegate: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}),
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	// this call should delegate
	resp, err := http.Get(server.URL + "/apis/groupName/foo")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected %v, got %v", http.StatusForbidden, resp.StatusCode)
	}

	// groupName still has no api services for it (like it was deleted), it should delegate
	resp, err = http.Get(server.URL + "/apis/groupName/")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected %v, got %v", http.StatusForbidden, resp.StatusCode)
	}

	// missing group should delegate still has no api services for it (like it was deleted)
	resp, err = http.Get(server.URL + "/apis/missing")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected %v, got %v", http.StatusForbidden, resp.StatusCode)
	}
}

func TestAPIGroup(t *testing.T) {
	tests := []struct {
		name        string
		group       string
		apiservices []*apiregistration.APIService
		expected    *metav1.APIGroup
	}{
		{
			name:  "sorting",
			group: "foo",
			apiservices: []*apiregistration.APIService{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v1.foo"},
					Spec: apiregistration.APIServiceSpec{
						Service: &apiregistration.ServiceReference{
							Namespace: "ns",
							Name:      "api",
						},
						Group:                "foo",
						Version:              "v1",
						GroupPriorityMinimum: 20,
						VersionPriority:      10,
					},
					Status: apiregistration.APIServiceStatus{
						Conditions: []apiregistration.APIServiceCondition{
							{Type: apiregistration.Available, Status: apiregistration.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v2.bar"},
					Spec: apiregistration.APIServiceSpec{
						Service: &apiregistration.ServiceReference{
							Namespace: "ns",
							Name:      "api",
						},
						Group:                "bar",
						Version:              "v2",
						GroupPriorityMinimum: 11,
					},
					Status: apiregistration.APIServiceStatus{
						Conditions: []apiregistration.APIServiceCondition{
							{Type: apiregistration.Available, Status: apiregistration.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v2.foo"},
					Spec: apiregistration.APIServiceSpec{
						Service: &apiregistration.ServiceReference{
							Namespace: "ns",
							Name:      "api",
						},
						Group:                "foo",
						Version:              "v2",
						GroupPriorityMinimum: 1,
						VersionPriority:      15,
					},
					Status: apiregistration.APIServiceStatus{
						Conditions: []apiregistration.APIServiceCondition{
							{Type: apiregistration.Available, Status: apiregistration.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v1.bar"},
					Spec: apiregistration.APIServiceSpec{
						Service: &apiregistration.ServiceReference{
							Namespace: "ns",
							Name:      "api",
						},
						Group:                "bar",
						Version:              "v1",
						GroupPriorityMinimum: 11,
					},
					Status: apiregistration.APIServiceStatus{
						Conditions: []apiregistration.APIServiceCondition{
							{Type: apiregistration.Available, Status: apiregistration.ConditionTrue},
						},
					},
				},
			},
			expected: &metav1.APIGroup{
				TypeMeta: metav1.TypeMeta{Kind: "APIGroup", APIVersion: "v1"},
				Name:     "foo",
				Versions: []metav1.GroupVersionForDiscovery{
					{
						GroupVersion: "foo/v2",
						Version:      "v2",
					},
					{
						GroupVersion: "foo/v1",
						Version:      "v1",
					},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{
					GroupVersion: "foo/v2",
					Version:      "v2",
				},
			},
		},
	}

	for _, tc := range tests {
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		handler := &apiGroupHandler{
			codecs:    apiserverSchema.Codecs,
			lister:    generatedClientListersV1.NewAPIServiceLister(indexer),
			groupName: "foo",
		}
		for _, o := range tc.apiservices {
			_ = indexer.Add(o)
		}

		server := httptest.NewServer(handler)
		//goland:noinspection GoDeferInLoop
		defer server.Close()

		resp, err := http.Get(server.URL + "/apis/" + tc.group)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			response, _ := httputil.DumpResponse(resp, true)
			t.Errorf("%s: %v", tc.name, string(response))
			continue
		}
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}

		actual := &metav1.APIGroup{}
		if err := runtime.DecodeInto(apiserverSchema.Codecs.UniversalDecoder(), bytes, actual); err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if !equality.Semantic.DeepEqual(tc.expected, actual) {
			t.Errorf("%s: %v", tc.name, cmp.Diff(tc.expected, actual))
			continue
		}
	}
}
