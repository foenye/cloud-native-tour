package main

import (
	"context"
	"encoding/json"
	"github.com/eonvon/cloud-native-tour/apiserver-from-scratch/cmd/foos"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscovery(t *testing.T) {
	server := httptest.NewServer(ServeMux())
	defer server.Close()
	kube := kubernetes.NewForConfigOrDie(&rest.Config{Host: server.URL})

	groupVersionResources, err := restmapper.GetAPIGroupResources(kube.DiscoveryClient)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(*groupVersionResources[0])

	discoveryRESTMapper := restmapper.NewDiscoveryRESTMapper(groupVersionResources)
	shortcutExpander := restmapper.NewShortcutExpander(discoveryRESTMapper, kube.DiscoveryClient, nil)

	tests := []struct {
		name                       string
		srcGroupVersionResource    schema.GroupVersionResource
		targetGroupVersionResource schema.GroupVersionResource
		targetGroupVersionKind     schema.GroupVersionKind
		mapperFn                   func() meta.RESTMapper
		wantErr                    bool
	}{
		{
			name:                    `When GroupVersionResource is ("", "", "fo") and RESTMapper is ShortcutExpander`,
			srcGroupVersionResource: schema.GroupVersionResource{Resource: "fo"},
			targetGroupVersionResource: schema.GroupVersionResource{Group: "hello.eonvon.github.io", Version: "v1",
				Resource: "foos"},
			targetGroupVersionKind: schema.GroupVersionKind{Group: "hello.eonvon.github.io", Version: "v1", Kind: "Foo"},
			mapperFn: func() meta.RESTMapper {
				return shortcutExpander
			},
			wantErr: false,
		},
		{
			name:                    `When GroupVersionResource is ("", "", "fo") and RESTMapper is DiscoveryRESTMapper`,
			srcGroupVersionResource: schema.GroupVersionResource{Resource: "fo"},
			targetGroupVersionResource: schema.GroupVersionResource{Group: "hello.eonvon.github.io", Version: "v1",
				Resource: "foos"},
			targetGroupVersionKind: schema.GroupVersionKind{Group: "hello.eonvon.github.io", Version: "v1", Kind: "Foo"},
			mapperFn: func() meta.RESTMapper {
				return discoveryRESTMapper
			},
			wantErr: true,
		},
		{
			name:                    `When GroupVersionResource is ("", "", "foo") and RESTMapper is DiscoveryRESTMapper`,
			srcGroupVersionResource: schema.GroupVersionResource{Resource: "foo"},
			targetGroupVersionResource: schema.GroupVersionResource{Group: "hello.eonvon.github.io", Version: "v1",
				Resource: "foos"},
			targetGroupVersionKind: schema.GroupVersionKind{Group: "hello.eonvon.github.io", Version: "v1", Kind: "Foo"},
			mapperFn: func() meta.RESTMapper {
				return discoveryRESTMapper
			},
			wantErr: false,
		},
		{
			name:                    `When GroupVersionResource is ("", "", "foos") and RESTMapper is DiscoveryRESTMapper`,
			srcGroupVersionResource: schema.GroupVersionResource{Resource: "foos"},
			targetGroupVersionResource: schema.GroupVersionResource{Group: "hello.eonvon.github.io", Version: "v1",
				Resource: "foos"},
			targetGroupVersionKind: schema.GroupVersionKind{Group: "hello.eonvon.github.io", Version: "v1", Kind: "Foo"},
			mapperFn: func() meta.RESTMapper {
				return discoveryRESTMapper
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			groupVersionKind, err := test.mapperFn().KindFor(test.srcGroupVersionResource)
			groupVersionResource, err1 := test.mapperFn().ResourceFor(test.srcGroupVersionResource)
			if (err != nil) != test.wantErr {
				t.Errorf("KindFor() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if (err1 != nil) != test.wantErr {
				t.Errorf("ResourcesFor() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			if test.wantErr {
				return
			}

			if groupVersionKind != test.targetGroupVersionKind {
				t.Errorf("KindFor() gotGroupVersionKind = %v, want %v", groupVersionKind,
					test.targetGroupVersionKind)
			}

			if groupVersionResource != test.targetGroupVersionResource {
				t.Errorf("ResourceFor() gotGroupVersionResource  = %v, want %v", groupVersionResource,
					test.targetGroupVersionResource)
			}

		})
	}
}

func TestFooAPIs(t *testing.T) {
	server := httptest.NewServer(ServeMux())
	defer server.Close()
	kube := kubernetes.NewForConfigOrDie(&rest.Config{Host: server.URL})

	restClient := kube.RESTClient()
	result := restClient.Verb(http.MethodGet).Prefix("apis", "hello.eonvon.github.io", "v1").
		Namespace(metav1.NamespaceDefault).Resource("foos").Name("bar").Do(context.Background())
	if err := result.Error(); err != nil {
		t.Fatalf("RESTGetFoo() error = %v", err)
	}
	if err := result.Into(&foos.Foo{}); err != nil {
		t.Fatalf("RESTGetFoo() restult convert into Foo error = %v", err)
	}

	createFooCommand, _ := json.Marshal(&foos.Foo{
		ObjectMeta: metav1.ObjectMeta{
			Name: "newFoo",
		},
		Spec: foos.FooSpec{
			Msg:         "hi newFoo",
			Description: "Create a new foo from test says hi 👏🏻",
		},
	})
	var created = foos.Foo{}
	err := restClient.Verb(http.MethodPost).Prefix("apis", "hello.eonvon.github.io", "v1").
		Namespace(metav1.NamespaceDefault).Resource("foos").Body(createFooCommand).Do(context.Background()).Into(&created)
	if err != nil {
		t.Fatalf("RESTCreateFoo() error=%v", err)
	}
	if created.Name != "newFoo" || created.Spec.Msg != "hi newFoo" ||
		created.Spec.Description != "Create a new foo from test says hi 👏🏻" {
		t.Fatalf("RESTCreateFoo() error=%v, want name=%s msg=%s description=%s, actual %v ", err,
			"newFoo", "hi newFoo", "Create a new foo from test says hi 👏🏻", created)
	}

	var patched = foos.Foo{}
	patchFooCommand, _ := json.Marshal(&foos.Foo{
		ObjectMeta: metav1.ObjectMeta{
			Name: "patchedFoo",
		},
		Spec: foos.FooSpec{
			Msg: "Ah ah ah",
		},
	})
	err = restClient.Verb(http.MethodPatch).Prefix("apis", "hello.eonvon.github.io", "v1").
		SetHeader("Content-Type", "application/strategic-merge-patch+json").
		Namespace(metav1.NamespaceDefault).Resource("foos").Name("newFoo").Body(patchFooCommand).
		Do(context.Background()).Into(&patched)
	if err != nil || patched.Spec.Msg != "Ah ah ah" {
		t.Fatalf("RESTReviseFoo() error = %v", err)
	}

	err = restClient.Verb(http.MethodDelete).Prefix("apis", "hello.eonvon.github.io", "v1").
		Namespace(metav1.NamespaceDefault).Resource("foos").Name("newFoo").Do(context.Background()).Into(&foos.Foo{})
	if err != nil {
		t.Fatalf("RESTDeleteFoo error = %v", err)
	}

}
