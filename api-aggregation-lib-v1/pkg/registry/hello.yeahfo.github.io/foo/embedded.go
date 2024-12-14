package foo

import (
	"context"
	"fmt"
	hellov1 "github.com/yeahfo/cloud-native-tour/api/hello.yeahfo.github.io/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage/names"
	"sync"
	"time"
)

var (
	_ rest.Getter               = &API{}
	_ rest.Lister               = &API{}
	_ rest.Scoper               = &API{}
	_ rest.Storage              = &API{}
	_ rest.KindProvider         = &API{}
	_ rest.CreaterUpdater       = &API{}
	_ rest.GracefulDeleter      = &API{}
	_ rest.CollectionDeleter    = &API{}
	_ rest.SingularNameProvider = &API{}
)

func NewEmbeddedStore() *API {
	return &API{
		store: map[string]*hellov1.Foo{
			"default/bar": {
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         metav1.NamespaceDefault,
					Name:              "bar",
					CreationTimestamp: metav1.Now(),
				},
				Spec: hellov1.FooSpec{
					Msg:         "Hello World!",
					Description: "Made in apiserver ontop k8s.io/apiserver library.",
				},
			},
		},
	}
}

type API struct {
	sync.RWMutex
	store map[string]*hellov1.Foo
}

// DeleteCollection implements rest.CollectionDeleter
func (api *API) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, _ *metav1.DeleteOptions,
	_ *internalversion.ListOptions) (runtime.Object, error) {
	namespace := request.NamespaceValue(ctx)

	var fooList hellov1.FooList

	api.Lock()
	defer api.Unlock()

	for _, foo := range api.store {
		if foo.Namespace == namespace {
			fooList.Items = append(fooList.Items, *foo)
		}
	}

	if deleteValidation != nil {
		if err := deleteValidation(ctx, &fooList); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
	}

	for _, foo := range api.store {
		if foo.Namespace == namespace {
			delete(api.store, formatNamespacedName(namespace, foo.Name))
		}
	}

	return &fooList, nil
}

// Delete implements rest.GracefulDeleter
func (api *API) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc,
	_ *metav1.DeleteOptions) (runtime.Object, bool, error) {
	namespacedName := getNamespacedName(ctx, name)

	api.Lock()
	defer api.Unlock()

	if foo, exists := api.store[namespacedName]; !exists {
		return nil, false, errors.NewNotFound(hellov1.Resource(hellov1.Plural), namespacedName)
	} else {
		if deleteValidation != nil {
			if err := deleteValidation(ctx, foo); err != nil {
				return nil, false, errors.NewBadRequest(err.Error())
			}
		}
		delete(api.store, namespacedName)
		return foo, true, nil
	}
}

// Create implements rest.Creater
func (api *API) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc,
	_ *metav1.CreateOptions) (runtime.Object, error) {
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

	api.Lock()
	defer api.Unlock()

	namespacedName = formatNamespacedName(namespace, name)
	if _, exists := api.store[namespacedName]; exists {
		return nil, errors.NewAlreadyExists(hellov1.Resource(hellov1.Plural), namespacedName)
	}

	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
	}

	api.store[namespacedName] = obj.(*hellov1.Foo)
	return obj, nil
}

// Update implements rest.Updater
func (api *API) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, _ bool,
	_ *metav1.UpdateOptions) (runtime.Object, bool, error) {
	namespace := request.NamespaceValue(ctx)
	namespacedName := formatNamespacedName(namespace, name)

	var (
		existingFoo, creatingFoo runtime.Object
		creating                 = false
		err                      error
	)

	api.Lock()
	defer api.Unlock()

	if existingFoo = api.store[namespacedName]; existingFoo.(*hellov1.Foo) == nil {
		creating = true
		creatingFoo = api.New()
		creatingFoo, err = objInfo.UpdatedObject(ctx, creatingFoo)
		if err != nil {
			return nil, false, errors.NewBadRequest(err.Error())
		}
	}

	if creating {
		creatingFoo, err = api.Create(ctx, creatingFoo, createValidation, nil)
		if err != nil {
			return nil, false, err
		}
		return creatingFoo, true, nil
	}

	updated, err := objInfo.UpdatedObject(ctx, existingFoo)
	if err != nil {
		return nil, false, errors.NewInternalError(err)
	}

	if updateValidation != nil {
		if err := updateValidation(ctx, updated, existingFoo); err != nil {
			return nil, false, errors.NewBadRequest(err.Error())
		}
	}

	api.store[namespacedName] = updated.(*hellov1.Foo)
	return updated, false, nil
}

// NewList implements rest.Lister
func (api *API) NewList() runtime.Object {
	return &hellov1.FooList{}
}

// List implements rest.Lister
func (api *API) List(ctx context.Context, _ *internalversion.ListOptions) (runtime.Object, error) {
	namespace := request.NamespaceValue(ctx)

	api.Lock()
	defer api.Unlock()

	var fooList hellov1.FooList

	for _, currentFoo := range api.store {
		if namespace == "" {
			fooList.Items = append(fooList.Items, *currentFoo)
		} else {
			if currentFoo.Namespace == namespace {
				fooList.Items = append(fooList.Items, *currentFoo)
			}
		}
	}
	return &fooList, nil
}

// ConvertToTable implements rest.Lister
func (api *API) ConvertToTable(_ context.Context, object runtime.Object, _ runtime.Object) (*metav1.Table, error) {
	var table metav1.Table

	table.ColumnDefinitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
		{Name: "Message", Type: "string", Format: "message", Description: "api message"},
		{Name: "Description", Type: "string", Format: "description", Description: "api message plus"},
	}

	switch typed := object.(type) {
	case *hellov1.Foo:
		table.ResourceVersion = typed.ResourceVersion
		addSomeFooToTable(&table, *typed)
	case *hellov1.FooList:
		table.ResourceVersion = typed.ResourceVersion
		table.Continue = typed.Continue
		addSomeFooToTable(&table, typed.Items...)
	default:
	}
	return &table, nil
}

// Get implements rest.Getter.
func (api *API) Get(ctx context.Context, name string, _ *metav1.GetOptions) (runtime.Object, error) {
	namespacedName := getNamespacedName(ctx, name)

	api.Lock()
	defer api.Unlock()

	if got, exists := api.store[namespacedName]; !exists {
		return nil, errors.NewNotFound(hellov1.Resource(hellov1.Plural), namespacedName)
	} else {
		return got, nil
	}
}

// GetSingularName implements rest.SingularNameProvider
func (api *API) GetSingularName() string {
	return hellov1.Resource(hellov1.Singular).Resource
}

// Kind implements rest.KindProvider
func (api *API) Kind() string {
	return hellov1.Kind
}

// NamespaceScoped implements rest.Scoper
func (api *API) NamespaceScoped() bool {
	return true
}

// New implements rest.Storage
func (api *API) New() runtime.Object {
	return &hellov1.Foo{}
}

// Destroy implements rest.Storage
func (api *API) Destroy() {
}

func getNamespacedName(ctx context.Context, name string) string {
	namespace := request.NamespaceValue(ctx)
	return formatNamespacedName(namespace, name)
}

func formatNamespacedName(namespace string, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func addSomeFooToTable(table *metav1.Table, fooList ...hellov1.Foo) {
	for _, foo := range fooList {
		ageColumn := "<unknown>"
		if creationTimestamp := foo.CreationTimestamp; !creationTimestamp.IsZero() {
			ageColumn = duration.HumanDuration(time.Since(creationTimestamp.Time))
		}
		table.Rows = append(table.Rows, metav1.TableRow{
			Cells:  []interface{}{foo.Name, ageColumn, foo.Spec.Msg, foo.Spec.Description},
			Object: runtime.RawExtension{Object: &foo},
		})
	}
}
