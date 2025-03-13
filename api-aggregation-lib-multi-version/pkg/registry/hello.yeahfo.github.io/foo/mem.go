package foo

import (
	"context"
	"fmt"
	"github.com/eonvon/cloud-native-tour/api-aggregation-lib-multi-version/pkg/api/hello.eonvon.github.io"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage/names"
	"sync"
)

// InMemory interfaces implementations
var (
	_ rest.Scoper               = &InMemory{}
	_ rest.Getter               = &InMemory{}
	_ rest.Lister               = &InMemory{}
	_ rest.Storage              = &InMemory{}
	_ rest.KindProvider         = &InMemory{}
	_ rest.CreaterUpdater       = &InMemory{}
	_ rest.GracefulDeleter      = &InMemory{}
	_ rest.CollectionDeleter    = &InMemory{}
	_ rest.ShortNamesProvider   = &InMemory{}
	_ rest.SingularNameProvider = &InMemory{}
)

type InMemory struct {
	sync.RWMutex
	repository map[string]*hello.Foo
}

func NewInMemory() *InMemory {
	return &InMemory{repository: map[string]*hello.Foo{
		metav1.NamespaceDefault + "/bar": {
			ObjectMeta: metav1.ObjectMeta{Namespace: metav1.NamespaceDefault, Name: "bar", CreationTimestamp: metav1.Now()},
			Spec: hello.FooSpec{
				Image: "busybox:1.36",
				Config: hello.FooConfig{
					Msg:         "hello world 👋",
					Description: "made in apiserver using library k8s.io/apiserver 👊",
				},
			},
		},
	}}
}

func (memory *InMemory) GetSingularName() string {
	return hello.Resource(hello.Singular).Resource
}

// ShortNames implements rest.ShortNamesProvider
func (memory *InMemory) ShortNames() []string {
	return []string{hello.ShortName}
}

// DeleteCollection implements rest.CollectionDeleter
func (memory *InMemory) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, _ *metav1.DeleteOptions, _ *internalversion.ListOptions) (runtime.Object, error) {
	namespace := request.NamespaceValue(ctx)

	var fooList hello.FooList

	memory.Lock()
	defer memory.Unlock()

	for _, foo := range memory.repository {
		if foo.Namespace == namespace {
			fooList.Items = append(fooList.Items, *foo)
		}
	}

	if deleteValidation != nil {
		if err := deleteValidation(ctx, &fooList); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
	}

	for _, foo := range memory.repository {
		if foo.Namespace == namespace {
			delete(memory.repository, getNamespacedName(ctx, foo.Name))
		}
	}

	return &fooList, nil
}

// Delete implements rest.GracefulDeleter
func (memory *InMemory) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, _ *metav1.DeleteOptions) (runtime.Object, bool, error) {
	namespacedName := getNamespacedName(ctx, name)

	memory.Lock()
	defer memory.Unlock()

	if foo, exists := memory.repository[namespacedName]; !exists {
		return nil, false, errors.NewNotFound(hello.Resource(hello.GroupName), namespacedName)
	} else {
		if deleteValidation != nil {
			if err := deleteValidation(ctx, foo); err != nil {
				return nil, false, errors.NewBadRequest(err.Error())
			}
		}
		delete(memory.repository, namespacedName)
		return foo, true, nil
	}
}

// New implements rest.CreaterUpdater
func (memory *InMemory) New() runtime.Object {
	return &hello.Foo{}
}

// Create implements rest.CreaterUpdater
func (memory *InMemory) Create(_ context.Context, obj runtime.Object, _ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
	var name, namespace, namespacedName string

	if objectMeta, err := meta.Accessor(obj); err != nil {
		return nil, errors.NewInternalError(err)
	} else {
		rest.FillObjectMetaSystemFields(objectMeta)
		if len(objectMeta.GetGenerateName()) > 0 && len(objectMeta.GetName()) == 0 {
			objectMeta.SetName(names.SimpleNameGenerator.GenerateName(objectMeta.GetGenerateName()))
		}
		name = objectMeta.GetName()
		namespace = objectMeta.GetNamespace()
	}

	memory.Lock()
	defer memory.Unlock()

	namespacedName = formatNamespacedName(namespace, name)
	if _, exists := memory.repository[namespacedName]; exists {
		return nil, errors.NewAlreadyExists(hello.Resource(hello.Plural), namespacedName)
	}

	memory.repository[namespacedName] = obj.(*hello.Foo)
	return obj, nil
}

// Update implements rest.CreaterUpdater
func (memory *InMemory) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, _ bool, _ *metav1.UpdateOptions) (runtime.Object, bool, error) {
	namespacedName := getNamespacedName(ctx, name)

	var (
		existingFoo, creatingFoo runtime.Object
		creating                 = false
		err                      error
	)

	memory.Lock()
	defer memory.Unlock()

	if existingFoo = memory.repository[namespacedName]; existingFoo.(*hello.Foo) == nil {
		creating = true
		creatingFoo = memory.New()
		creatingFoo, err = objInfo.UpdatedObject(ctx, creatingFoo)
		if err != nil {
			return nil, false, errors.NewBadRequest(err.Error())
		}
	}

	if creating {
		creatingFoo, err = memory.Create(ctx, creatingFoo, createValidation, nil)
		if err != nil {
			return nil, false, err
		}
		return creatingFoo, true, nil
	}

	updatedFoo, err := objInfo.UpdatedObject(ctx, existingFoo)
	if err != nil {
		return nil, false, errors.NewInternalError(err)
	}

	if updateValidation != nil {
		if err = updateValidation(ctx, updatedFoo, existingFoo); err != nil {
			return nil, false, errors.NewBadRequest(err.Error())
		}
	}

	memory.repository[namespacedName] = updatedFoo.(*hello.Foo)
	return updatedFoo, false, nil
}

// Destroy implements rest.Storage
func (memory *InMemory) Destroy() {

}

// Kind implements rest.KindProvider
func (memory *InMemory) Kind() string {
	return "Foo"
}

// NewList implements rest.Lister
func (memory *InMemory) NewList() runtime.Object {
	return &hello.FooList{}
}

// List implements rest.Lister
func (memory *InMemory) List(ctx context.Context, _ *internalversion.ListOptions) (runtime.Object, error) {
	namespace := request.NamespaceValue(ctx)

	memory.Lock()
	defer memory.Unlock()

	var fooList hello.FooList

	for _, foo := range memory.repository {
		if namespace == "" {
			fooList.Items = append(fooList.Items, *foo)
		} else {
			if foo.Namespace == namespace {
				fooList.Items = append(fooList.Items, *foo)
			}
		}
	}
	return &fooList, nil
}

// ConvertToTable implements rest.Lister
func (memory *InMemory) ConvertToTable(_ context.Context, object runtime.Object, _ runtime.Object) (*metav1.Table, error) {
	var table metav1.Table

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Status", Type: "string", Format: "status", Description: "status of where the Foo is in its lifecycle"},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Message", Type: "string", Format: "message", Description: "foo message", Priority: 1},              // kubectl -o wide
		{Name: "Description", Type: "string", Format: "description", Description: "foo message plus", Priority: 1}, // kubectl -o wide
	}

	switch typed := object.(type) {
	case *hello.Foo:
		table.ResourceVersion = typed.ResourceVersion
		addSomeFooToTable(&table, *typed)
	case *hello.FooList:
		table.ResourceVersion = typed.ResourceVersion
		table.Continue = typed.Continue
		addSomeFooToTable(&table, typed.Items...)
	default:
	}
	return &table, nil
}

// Get implements rest.Getter interface
func (memory *InMemory) Get(ctx context.Context, name string, _ *metav1.GetOptions) (runtime.Object, error) {
	namespacedName := getNamespacedName(ctx, name)

	memory.Lock()
	defer memory.Unlock()

	if foo, exists := memory.repository[namespacedName]; !exists {
		return nil, errors.NewNotFound(hello.Resource(hello.Plural), namespacedName)
	} else {
		return foo, nil
	}
}

// NamespaceScoped implements rest.Scoper
func (memory *InMemory) NamespaceScoped() bool {
	return true
}

func getNamespacedName(ctx context.Context, name string) string {
	namespace := request.NamespaceValue(ctx)
	return formatNamespacedName(namespace, name)
}

func formatNamespacedName(namespace string, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}
