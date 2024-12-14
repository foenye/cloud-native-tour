package foos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/yeahfo/cloud-native-tour/apiserver-from-scratch/pkg/helper"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeRuntime "k8s.io/apimachinery/pkg/runtime"
	kubeDurationUtils "k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	fooRepository = make(map[string]Foo)
	fooLock       sync.RWMutex
)

type namespaceContextKey string

const namespaced = namespaceContextKey("namespace")

type Foo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FooSpec `json:"spec"`
}

func (foo *Foo) DeepCopyObject() kubeRuntime.Object {
	copied := *foo
	return &copied
}

func (foo *Foo) toTableRow() []metav1.TableRow {
	ageColumn := "<unknown>"

	if creationTimestamp := foo.CreationTimestamp; !creationTimestamp.IsZero() {
		ageColumn = kubeDurationUtils.HumanDuration(time.Since(creationTimestamp.Time))
	}
	return []metav1.TableRow{
		{
			Object: kubeRuntime.RawExtension{Object: foo},
			Cells:  []interface{}{foo.Name, ageColumn, foo.Spec.Msg, foo.Spec.Description},
		},
	}
}

type FooSpec struct {
	// Msg says hello world!
	Msg string `json:"msg"`
	// Description of FooSpec details.
	Description string `json:"description"`
}

type FooList struct {
	metav1.TypeMeta   `json:"inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Items []Foo `json:"items"`
}

func (fooList *FooList) toTableRows() (tableRows []metav1.TableRow) {
	for idx := range fooList.Items {
		tableRows = append(tableRows, fooList.Items[idx].toTableRow()...)
	}
	return tableRows
}

func init() {
	fooRepository["default/bar"] = Foo{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hello.yeahfo.github.io/v1",
			Kind:       "Foo",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "bar",
			Namespace:         metav1.NamespaceDefault,
			CreationTimestamp: metav1.Now(),
		},
		Spec: FooSpec{
			Msg:         "Hello World!",
			Description: "APIServer-from-scratch says '👋 hello world 👋'",
		},
	}
}

var FooColumnDefinitions = []metav1.TableColumnDefinition{
	{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
	{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	{Name: "Message", Type: "string", Format: "message", Description: "foo message"},
	{Name: "Description", Type: "string", Format: "description", Description: "foo message plus", Priority: 1}, // kubectl -o wide
}

// tryConvert2Table application/json;as=Table;v=v1;g=meta.k8s.io,application/json;as=Table;v=v1beta1;g=meta.k8s.io,application/json
func tryConvert2Table(obj interface{}, acceptedContentType string) interface{} {
	if strings.Contains(acceptedContentType, "application/json") && strings.Contains(acceptedContentType, "as=Table") {
		switch typedObj := obj.(type) {
		case Foo:
			return metav1.Table{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "meta.k8s.io/v1",
					Kind:       "Table",
				},
				ColumnDefinitions: FooColumnDefinitions,
				Rows:              typedObj.toTableRow(),
			}
		case FooList:
			return metav1.Table{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "meta.k8s.io/v1",
					Kind:       "Table",
				},
				ColumnDefinitions: FooColumnDefinitions,
				Rows:              typedObj.toTableRows(),
			}
		default:
			return obj
		}
	}
	return obj
}

func FooHandlers(responseWriter http.ResponseWriter, request *http.Request) {
	requestPath := request.URL.Path
	namespacedResource := strings.TrimPrefix(requestPath, "/apis/hello.yeahfo.github.io/v1/namespaces/")
	if namespacedResource == requestPath &&
		requestPath == "/apis/hello.yeahfo.github.io/v1/foos" {
		getAllFoos(responseWriter, request)
		return
	}

	parts := strings.Split(namespacedResource, "/")
	if len(parts) == 2 {
		request = request.WithContext(context.WithValue(request.Context(), namespaced, parts[0]))
		switch request.Method {
		case http.MethodGet:
			GetAllFoosInNamespace(responseWriter, request)
		case http.MethodPost:
			CreateFoo(responseWriter, request)
		default:
			responseWriter.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else if len(parts) == 3 {
		request = request.WithContext(context.WithValue(request.Context(), namespaced, parts[0]))
		name := parts[2]
		switch request.Method {
		case http.MethodGet:
			GetFoo(responseWriter, request, name)
		case http.MethodPut:
			UpdateFoo(responseWriter, request, name)
		case http.MethodPatch:
			ReviseFoo(responseWriter, request, name)
		case http.MethodDelete:
			DeleteFoo(responseWriter, request, name)
		default:
			responseWriter.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		responseWriter.WriteHeader(http.StatusNotFound)
	}
}

// DeleteFoo swag doc
// @Summary      Delete a Foo Object
// @Description  Delete a Foo Object by name in some Namespace
// @Tags         foos
// @Produce      json
// @Param        namespace	path	string  true  "Namepsace"
// @Param        name	path	string  true  "Resource Name"
// @Success      200  {object}  Foo "deleted"
// @Router       /apis/hello.yeahfo.github.io/v1/namespaces/{namespace}/foos/{name} [delete]
func DeleteFoo(writer http.ResponseWriter, request *http.Request, name string) {
	namespace := request.Context().Value(namespaced)
	namespacedName := fmt.Sprintf("%s/%s", namespace, name)

	fooLock.Lock()
	defer fooLock.Unlock()

	if foo, exists := fooRepository[namespacedName]; !exists { // not exists
		helper.WriteErrorStatus(writer, namespacedName, http.StatusNotFound, "")
		return
	} else {
		delete(fooRepository, namespacedName)
		now := metav1.Now()
		var noWait int64 = 0
		foo.DeletionTimestamp = &now
		foo.DeletionGracePeriodSeconds = &noWait
		helper.RenderJSON(writer, foo) // follow official API, return the deleted object
	}
}

// ReviseFoo swag doc
// @Summary      partially update the specified Foo
// @Description  partially update the specified Foo
// @Tags         foos
// @Consume      json
// @Produce      json
// @Param        namespace	path	string  true  "Namepsace"
// @Param        name	path	string  true  "Resource Name"
// @Success      200  {object}  Foo "OK"
// @Router       /apis/hello.yeahfo.github.io/v1/namespaces/{namespace}/foos/{name} [patch]
func ReviseFoo(writer http.ResponseWriter, request *http.Request, name string) {
	namespace := request.Context().Value(namespaced)
	namespacedName := fmt.Sprintf("%s/%s", namespace, name)

	reviseFooBytes, err := io.ReadAll(request.Body)
	if err != nil {
		helper.WriteErrorStatus(writer, namespacedName, http.StatusBadRequest, "")
		return
	}

	fooLock.Lock()
	defer fooLock.Unlock()

	rawFoo, exists := fooRepository[namespacedName]
	if !exists { // not exists
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	rawFooEncoded, _ := json.Marshal(rawFoo)
	var revisedFooEncoded []byte

	switch request.Header.Get("Content-Type") {
	case "application/merge-patch+json":
		revisedFooEncoded, err = jsonpatch.MergePatch(rawFooEncoded, reviseFooBytes)
		if err != nil {
			helper.WriteErrorStatus(writer, namespacedName, http.StatusBadRequest, err.Error())
			return
		}
	case "application/json-patch+json":
		reviser, err := jsonpatch.DecodePatch(reviseFooBytes)
		if err != nil {
			helper.WriteErrorStatus(writer, namespacedName, http.StatusBadRequest, err.Error())
			return
		}
		revisedFooEncoded, err = reviser.Apply(rawFooEncoded)
		if err != nil {
			helper.WriteErrorStatus(writer, namespacedName, http.StatusBadRequest, err.Error())
			return
		}
	case "application/strategic-merge-patch+json":
		var schema = rawFoo
		unstructuredRawFoo, _ := kubeRuntime.DefaultUnstructuredConverter.ToUnstructured(&rawFoo)
		var revisedHolder map[string]interface{}

		if err := json.Unmarshal(reviseFooBytes, &revisedHolder); err != nil {
			helper.WriteErrorStatus(writer, namespacedName, http.StatusBadRequest, err.Error())
			return
		}

		if revisedHolder, err := strategicpatch.StrategicMergeMapPatch(unstructuredRawFoo, revisedHolder,
			schema); err != nil {
			helper.WriteErrorStatus(writer, namespacedName, http.StatusBadRequest, err.Error())
			return
		} else {
			var theFoo Foo
			if err = kubeRuntime.DefaultUnstructuredConverter.FromUnstructuredWithValidation(revisedHolder,
				&theFoo, false); err != nil {
				helper.WriteErrorStatus(writer, namespacedName, http.StatusBadRequest, err.Error())
				return
			} else {
				revisedFooEncoded, _ = json.Marshal(theFoo)
			}
		}
	default:
		writer.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	decoder := json.NewDecoder(bytes.NewReader(revisedFooEncoded))
	decoder.DisallowUnknownFields()

	var foo Foo
	if err := decoder.Decode(&foo); err != nil {
		helper.WriteErrorStatus(writer, namespacedName, http.StatusBadRequest, err.Error())
		return
	}
	fooRepository[namespacedName] = foo
	helper.RenderJSON(writer, foo)

}

// UpdateFoo swag doc
// @Summary      Replace a Foo Object
// @Description  Replace a Foo Object by Creation or Update
// @Tags         foos
// @Consume      json
// @Produce      json
// @Param        namespace	path	string  true  "Namepsace"
// @Param        name	path	string  true  "Resource Name"
// @Success      201  {object}  Foo	"created"
// @Success      200  {object}  Foo "updated"
// @Router       /apis/hello.yeahfo.github.io/v1/namespaces/{namespace}/foos/{name} [put]
func UpdateFoo(writer http.ResponseWriter, request *http.Request, name string) {
	namespace := request.Context().Value(namespaced)
	namespacedName := fmt.Sprintf("%s/%s", namespace, name)

	requestBodyDecoder := json.NewDecoder(request.Body)
	requestBodyDecoder.DisallowUnknownFields()

	var foo Foo
	if err := requestBodyDecoder.Decode(&foo); err != nil {
		helper.WriteErrorStatus(writer, namespacedName, http.StatusBadRequest, err.Error())
		return
	}

	fooLock.Lock()
	defer fooLock.Unlock()

	if _, exists := fooRepository[namespacedName]; !exists { // not exists
		writer.WriteHeader(http.StatusCreated)
	}
	fooRepository[namespacedName] = foo
	helper.RenderJSON(writer, foo) // follow official API, return the replacement
}

// CreateFoo swag doc
// @Summary      Create a Foo Object
// @Description  Create a Foo Object
// @Tags         foos
// @Consume      json
// @Produce      json
// @Param        namespace	path	string  true  "Namepsace"
// @Success      201  {object}  Foo
// @Router       /apis/hello.yeahfo.github.io/v1/namespaces/{namespace}/foos [post]
func CreateFoo(writer http.ResponseWriter, request *http.Request) {
	requestBodyDecoder := json.NewDecoder(request.Body)
	requestBodyDecoder.DisallowUnknownFields()

	var foo Foo
	if err := requestBodyDecoder.Decode(&foo); err != nil {
		helper.WriteErrorStatus(writer, "", http.StatusBadRequest, err.Error())
		return
	}

	foo.APIVersion = "hello.yeahfo.github.io/v1"
	foo.Kind = "Foo"
	foo.CreationTimestamp = metav1.Now()
	namespace := request.Context().Value(namespaced)
	foo.Namespace = namespace.(string)

	fooLock.Lock()
	defer fooLock.Unlock()

	namespacedName := fmt.Sprintf("%s/%s", namespace, foo.Name)
	if _, exists := fooRepository[namespacedName]; exists { // already exists
		helper.WriteErrorStatus(writer, namespacedName, http.StatusConflict, "")
		return
	}

	fooRepository[namespacedName] = foo
	writer.WriteHeader(http.StatusCreated)
	helper.RenderJSON(writer, foo)
}

// GetFoo swag doc
// @Summary      Get one Foo Object
// @Description  Get one Foo by Resource Name
// @Tags         foos
// @Produce      json
// @Param        namespace	path	string  true  "Namepsace"
// @Param        name	path	string  true  "Resource Name"
// @Success      200  {object}  Foo
// @Router       /apis/hello.yeahfo.github.io/v1/namespaces/{namespace}/foos/{name} [get]
func GetFoo(responseWriter http.ResponseWriter, request *http.Request, name string) {
	namespace := request.Context().Value(namespaced)
	namespacedName := fmt.Sprintf("%s/%s", namespace, name)

	fooLock.Lock()
	defer fooLock.Unlock()

	foo, exists := fooRepository[namespacedName]
	if !exists { // not exists
		helper.WriteErrorStatus(responseWriter, namespacedName, http.StatusNotFound, "")
		return
	}

	helper.RenderJSON(responseWriter, tryConvert2Table(foo, request.Header.Get("Accept")))
}

// GetAllFoos swag doc
// @Summary      List all Foos
// @Description  List all Foos
// @Tags         foos
// @Produce      json
// @Success      200  {object}  FooList
// @Router       /apis/hello.yeahfo.github.io/v1/foos [get]
func getAllFoos(responseWriter http.ResponseWriter, request *http.Request) {
	fooList := FooList{
		TypeMeta: metav1.TypeMeta{APIVersion: "hello.yeahfo.github.io/v1", Kind: "FooList"},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "fooList",
			CreationTimestamp: metav1.Now(),
		},
	}

	fooLock.Lock()
	defer fooLock.Unlock()

	for _, foo := range fooRepository {
		fooList.Items = append(fooList.Items, foo)
	}
	helper.RenderJSON(responseWriter, tryConvert2Table(fooList, request.Header.Get("Accept")))
}

// GetAllFoosInNamespace swag doc foo
// @Summary      List all Foos in some namespace
// @Description  List all Foos in some namespace
// @Tags         foos
// @Produce      json
// @Param        namespace	path	string  true  "Namepsace"
// @Success      200  {object}  FooList
// @Router       /apis/hello.yeahfo.github.io/v1/namespaces/{namespace}/foos [get]
func GetAllFoosInNamespace(responseWriter http.ResponseWriter, request *http.Request) {
	fooList := FooList{
		TypeMeta: metav1.TypeMeta{APIVersion: "hello.yeahfo.github.io/v1", Kind: "FooList"},
	}

	fooLock.Lock()
	defer fooLock.Unlock()

	namespace := request.Context().Value(namespaced)
	for _, foo := range fooRepository {
		if foo.Namespace == namespace {
			fooList.Items = append(fooList.Items, foo)
		}
	}
	helper.RenderJSON(responseWriter, tryConvert2Table(fooList, request.Header.Get("Accept")))
}
