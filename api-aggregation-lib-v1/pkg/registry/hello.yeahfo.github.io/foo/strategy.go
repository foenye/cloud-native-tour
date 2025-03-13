package foo

import (
	"context"
	"fmt"
	hellov1 "github.com/eonvon/cloud-native-tour/api/hello.eonvon.github.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
)

var (
	// strategy implements rest.TableConvertor
	_ rest.TableConvertor = strategy{}
	// strategy implements rest.RESTCreateUpdateStrategy
	_ rest.RESTCreateUpdateStrategy = strategy{}
)

type strategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// newStrategy creates and returns a foo strategy instance
func newStrategy(typer runtime.ObjectTyper) strategy {
	return strategy{typer, names.SimpleNameGenerator}
}

// getAttributes returns labels.Set, fields.Set, and error in case the given runtime.Object is not a Fischer.
func getAttributes(object runtime.Object) (labels.Set, fields.Set, error) {
	foo, instanceOfFoo := object.(*hellov1.Foo)

	if !instanceOfFoo {
		return nil, nil, fmt.Errorf("given object is not a Fischer")
	}
	return foo.ObjectMeta.Labels, selectableFields(foo), nil
}

func selectableFields(foo *hellov1.Foo) fields.Set {
	return generic.ObjectMetaFieldsSet(&foo.ObjectMeta, true)
}

// matchFoo is the filter used by the generic etcd backend to watch events from etcd to clients of the apiserver only
// interested in specific labels/fields.
func matchFoo(labelSelector labels.Selector, fieldSelector fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    labelSelector,
		Field:    fieldSelector,
		GetAttrs: getAttributes,
	}
}

func (strategy) ConvertToTable(_ context.Context, object runtime.Object, _ runtime.Object) (*metav1.Table, error) {
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

func (strategy) PrepareForCreate(_ context.Context, _ runtime.Object) {
}

func (strategy) Validate(_ context.Context, obj runtime.Object) field.ErrorList {
	_ = obj.(*hellov1.Foo)
	return nil
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (strategy) WarningsOnCreate(_ context.Context, _ runtime.Object) []string {
	return nil
}

func (strategy) Canonicalize(_ runtime.Object) {
}

func (strategy) NamespaceScoped() bool {
	return true
}

func (strategy) AllowCreateOnUpdate() bool {
	return false
}

func (strategy) PrepareForUpdate(_ context.Context, _, _ runtime.Object) {
}

func (strategy) ValidateUpdate(_ context.Context, _, _ runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (strategy) WarningsOnUpdate(_ context.Context, _, _ runtime.Object) []string {
	return nil
}

func (strategy) AllowUnconditionalUpdate() bool {
	return false
}
