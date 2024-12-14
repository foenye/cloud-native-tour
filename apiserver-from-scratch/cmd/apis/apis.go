package apis

import (
	"embed"
	"encoding/json"
	"github.com/yeahfo/cloud-native-tour/apiserver-from-scratch/pkg/helper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"net/http"
	"strings"
)

var apiDiscoveries = `{
	"apiVersion": "apidiscovery.k8s.io/v2beta1",
	"kind": "APIGroupDiscoveryList",
	"metadata": {},
	"items": [
	  {
		"metadata": {
		  "name": "hello.yeahfo.github.io"
		},
		"versions": [
		  {
			"version": "v1",
			"resources": [
			  {
				"resource": "foos",
				"responseKind": {
				  "group": "hello.yeahfo.github.io",
				  "kind": "Foo",
				  "version": "v1"
				},
				"scope": "Namespaced",
				"shortNames": [
				  "fo"
				],
				"singularResource": "foo",
				"verbs": [
				  "delete",
				  "get",
				  "list",
				  "patch",
				  "create",
				  "update"
				]
			  }
			]
		  }
		]
	  }
	]
}
`

var apis = metav1.APIGroupList{
	TypeMeta: metav1.TypeMeta{
		Kind:       "APIGroupList",
		APIVersion: "v1",
	},
	Groups: []metav1.APIGroup{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "APIGroup",
				APIVersion: "v1",
			},
			Name: "hello.yeahfo.github.io",
			Versions: []metav1.GroupVersionForDiscovery{
				{
					GroupVersion: "hello.yeahfo.github.io/v1",
					Version:      "v1",
				},
			},
			PreferredVersion: metav1.GroupVersionForDiscovery{GroupVersion: "hello.yeahfo.github.io/v1", Version: "v1"},
		},
	},
}

// APIs List APIGroups
//
//	@Summary		List all APIGroups of this apiserver
//	@Description	List all APIGroups of this apiserver
//	@Produce		json
//	@Success		200	{object} metav1.APIGroupList
//	@Router			/apis [get]
func APIs(responseWriter http.ResponseWriter, request *http.Request) {
	var groupVersionKind [3]string
	for _, acceptPart := range strings.Split(request.Header.Get("Accept"), ";") {
		if groupVersionKindPart := strings.Split(acceptPart, "="); len(groupVersionKindPart) == 2 {
			switch groupVersionKindPart[0] {
			case "g":
				groupVersionKind[0] = groupVersionKindPart[1]
			case "v":
				groupVersionKind[1] = groupVersionKindPart[1]
			case "as":
				groupVersionKind[2] = groupVersionKindPart[1]
			}
		}
	}

	if groupVersionKind[0] == "apidiscovery.k8s.io" && groupVersionKind[2] == "APIGroupDiscoveryList" {
		responseWriter.Header().Set("Content-Type", "application/json;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList")
		_, _ = responseWriter.Write([]byte(apiDiscoveries))
	} else {
		responseWriter.Header().Set("Content-Type", "application/json")
		helper.RenderJSON(responseWriter, apis)
	}
}

// APIGroup GET APIGroupHelloV1
//
//	@Summary		Get APIGroupHelloV1 info of 'hello.yeahfo.github.io'
//	@Description	Get APIGroupHelloV1 'hello.yeahfo.github.io' detail, including version list and preferred version
//	@Produce		json
//	@Success		200	{object} metav1.APIGroup
//	@Router			/apis/hello.yeahfo.github.io [get]
func APIGroup(responseWriter http.ResponseWriter, _ *http.Request) {
	responseWriter.Header().Set("Content-Type", "application/json")
	helper.RenderJSON(responseWriter, apis.Groups[0])
}

var apiResourceList = metav1.APIResourceList{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "APIResourceList",
	},
	GroupVersion: "hello.yeahfo.github.io/v1",
	APIResources: []metav1.APIResource{
		{
			Kind:         "Foo",
			Name:         "foos",
			ShortNames:   []string{"fo"},
			SingularName: "foo",
			Namespaced:   true,
			Categories:   []string{"all"},
			Verbs: metav1.Verbs([]string{
				"create",
				"delete",
				"get",
				"list",
				"update",
				"patch",
			}),
		},
	},
}

const v1Resources = `{
	"kind": "APIResourceList",
	"apiVersion": "v1",
	"groupVersion": "hello.yeahfo.github.io/v1",
	"resources": [
	  {
		"name": "foos",
		"singularName": "foo",
		"namespaced": true,
		"kind": "Foo",
		"verbs": [
		  "create",
		  "delete",
		  "get",
		  "list",
		  "update",
		  "patch"
		],
		"shortNames": [
		  "fo"
		],
		"categories": [
		  "all"
		]
	  }
	]
}`

// APIGroupV1Resources Get APIGroupV1Resources
//
//	@Summary		Get APIGroupHelloV1Resources for group version 'hello.yeahfo.github.io/v1'
//	@Description	List APIResource Info about group version 'hello.yeahfo.github.io/v1'
//	@Produce		json
//	@Success		200	{string} APIGroupV1Resources
//	@Router			/apis/hello.yeahfo.github.io/v1 [get]
func APIGroupV1Resources(responseWriter http.ResponseWriter, _ *http.Request) {
	responseWriter.Header().Set("Content-Type", "application/json")
	if selector := rand.IntnRange(0, 1); selector == 0 {
		objEncoded, _ := json.Marshal(apiResourceList)
		_, _ = responseWriter.Write(objEncoded)
	} else {
		_, _ = responseWriter.Write([]byte(v1Resources))
	}
}

//go:embed docs/*
var embedFS embed.FS

// OpenapiV2 Get OpenAPI Spec v2 doc
//
//	@Summary		Get OpenAPI Spec v2 doc of this server
//	@Description	Get OpenAPI Spec v2 doc of this server
//	@Produce		json
//	@Produce		application/com.github.proto-openapi.spec.v2@v1.0+protobuf
//	@Success		200	{string} swagger.json
//	@Router			/openapi/v2 [get]
func OpenapiV2(responseWriter http.ResponseWriter, request *http.Request) {
	openapiJSONBytes, _ := embedFS.ReadFile("docs/swagger.json")

	// 😭 kubectl (v1.26.2, v1.27.1 ...) api discovery module (which fetch /openapi/v2, /openapi/v3)
	//    only accept application/com.github.proto-openapi.spec.v2@v1.0+protobuf
	acceptHeader := request.Header.Get("Accept")
	if !strings.Contains(acceptHeader, "application/json") && strings.Contains(acceptHeader, "protobuf") {
		responseWriter.Header().Set("Content-Type", "application/com.github.proto-openapi.spec.v2.v1.0+protobuf")
		if protobufBytes, err := helper.ToProtoBinary(openapiJSONBytes); err != nil {
			responseWriter.Header().Set("Content-Type", "application/json")
			helper.WriteErrorStatus(responseWriter, "", http.StatusInternalServerError, err.Error())
			return
		} else {
			_, _ = responseWriter.Write(protobufBytes)
			return
		}
	}

	// 😄 kube apiserver aggregation module accept application/json
	responseWriter.Header().Set("Content-Type", "application/json")
	_, _ = responseWriter.Write(openapiJSONBytes)
}
